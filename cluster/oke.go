package cluster

import (
	"fmt"

	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	oracleClusterManager "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster/manager"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	secretOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	"github.com/banzaicloud/pipeline/secret"
)

// OKECluster struct for OKE cluster
type OKECluster struct {
	modelCluster *model.ClusterModel
	APIEndpoint  string
	CommonClusterBase
}

// CreateOKEClusterFromModel creates ClusterModel struct from model
func CreateOKEClusterFromModel(clusterModel *model.ClusterModel) (*OKECluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	okeCluster := OKECluster{
		modelCluster: clusterModel,
	}
	return &okeCluster, nil
}

// CreateOKEClusterFromRequest creates ClusterModel struct from the request
func CreateOKEClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint) (*OKECluster, error) {
	log.Debug("Create ClusterModel struct from the request")

	var oke OKECluster

	Model, err := modelOracle.CreateModelFromCreateRequest(request)
	if err != nil {
		return &oke, err
	}

	oke.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Oracle:         Model,
	}

	return &oke, nil
}

// CreateCluster creates a new cluster
func (o *OKECluster) CreateCluster() error {

	log.Info("Start creating Oracle cluster")

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	return cm.ManageOKECluster(&o.modelCluster.Oracle)
}

// UpdateCluster updates the cluster
func (o *OKECluster) UpdateCluster(r *pkgCluster.UpdateClusterRequest, userId uint) error {

	// todo add userid to nodes
	model, err := modelOracle.CreateModelFromUpdateRequest(o.modelCluster.Oracle, r)
	if err != nil {
		return err
	}

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	err = cm.ManageOKECluster(&model)
	if err != nil {
		return err
	}

	// remove node pools from model which are marked for deleting
	nodePools := make([]*modelOracle.NodePool, 0)
	for _, np := range model.NodePools {
		if !np.Delete {
			nodePools = append(nodePools, np)
		}
	}

	model.NodePools = nodePools
	o.modelCluster.Oracle = model

	return err
}

// DeleteCluster deletes cluster
func (o *OKECluster) DeleteCluster() error {

	// mark cluster model to deleting
	o.modelCluster.Oracle.Delete = true

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	return cm.ManageOKECluster(&o.modelCluster.Oracle)
}

//Persist save the cluster model
func (o *OKECluster) Persist(status, statusMessage string) error {

	return o.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (o *OKECluster) DownloadK8sConfig() ([]byte, error) {

	oci, err := o.GetOCI()
	if err != nil {
		return nil, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	return ce.GetK8SConfig(o.modelCluster.Oracle.OCID)
}

//GetName returns the name of the cluster
func (o *OKECluster) GetName() string {
	return o.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (o *OKECluster) GetType() string {
	return pkgCluster.Oracle
}

//GetStatus gets cluster status
func (o *OKECluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	return &pkgCluster.GetClusterStatusResponse{
		Status:        o.modelCluster.Status,
		StatusMessage: o.modelCluster.StatusMessage,
		Name:          o.modelCluster.Name,
		Location:      o.modelCluster.Location,
		Cloud:         pkgCluster.Oracle,
		ResourceID:    o.GetID(),
		NodePools:     nil,
	}, nil
}

//GetID returns the specified cluster id
func (o *OKECluster) GetID() uint {
	return o.modelCluster.ID
}

//GetModel returns the whole clusterModel
func (o *OKECluster) GetModel() *model.ClusterModel {
	return o.modelCluster
}

//CheckEqualityToUpdate validates the update request
func (o *OKECluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	return nil
}

//AddDefaultsToUpdate adds defaults to update request
func (o *OKECluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (o *OKECluster) GetAPIEndpoint() (string, error) {

	oci, err := o.GetOCI()
	if err != nil {
		return o.APIEndpoint, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return o.APIEndpoint, err
	}

	cluster, err := ce.GetCluster(&o.modelCluster.Oracle.OCID)
	if err != nil {
		return o.APIEndpoint, err
	}

	o.APIEndpoint = fmt.Sprintf("https://%s", *cluster.Endpoints.Kubernetes)

	return o.APIEndpoint, nil
}

// DeleteFromDatabase deletes model from the database
func (o *OKECluster) DeleteFromDatabase() error {
	err := o.modelCluster.Delete()
	if err != nil {
		return err
	}

	err = o.modelCluster.Oracle.Cleanup()
	if err != nil {
		return err
	}

	o.modelCluster = nil
	return nil
}

// GetOrganizationId gets org where the cluster belongs
func (o *OKECluster) GetOrganizationId() uint {
	return o.modelCluster.OrganizationId
}

//GetSecretId retrieves the secret id
func (o *OKECluster) GetSecretId() string {
	return o.modelCluster.SecretId
}

//GetSshSecretId retrieves the ssh secret id
func (o *OKECluster) GetSshSecretId() string {
	return o.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (o *OKECluster) SaveSshSecretId(sshSecretId string) error {
	return o.modelCluster.UpdateSshSecret(sshSecretId)
}

//CreateClusterFromModel creates the cluster from the model
func CreateClusterFromModel(clusterModel *model.ClusterModel) (*OKECluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	Cluster := OKECluster{
		modelCluster: clusterModel,
	}
	return &Cluster, nil
}

// UpdateStatus updates cluster status in database
func (o *OKECluster) UpdateStatus(status, statusMessage string) error {
	return o.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (o *OKECluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	status, err := o.GetStatus()
	if err != nil {
		return nil, err
	}

	// todo needs to add other fields
	return &pkgCluster.DetailsResponse{
		Name: status.Name,
		Id:   status.ResourceID,
	}, nil
}

// ValidateCreationFields validates all field
func (o *OKECluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

	cm, err := o.GetClusterManager()
	if err != nil {
		return err
	}

	return cm.ValidateModel(&o.modelCluster.Oracle)
}

// GetSecretWithValidation returns secret from vault
func (o *OKECluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return o.CommonClusterBase.getSecret(o)
}

// GetSshSecretWithValidation returns ssh secret from vault
func (o *OKECluster) GetSshSecretWithValidation() (*secret.SecretItemResponse, error) {
	return o.CommonClusterBase.getSecret(o)
}

// SaveConfigSecretId saves the config secret id in database
func (o *OKECluster) SaveConfigSecretId(configSecretId string) error {
	return o.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (o *OKECluster) GetConfigSecretId() string {
	return o.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (o *OKECluster) GetK8sConfig() ([]byte, error) {
	return o.DownloadK8sConfig()
}

// ReloadFromDatabase load cluster from DB
func (o *OKECluster) ReloadFromDatabase() error {
	return o.modelCluster.ReloadFromDatabase()
}

// GetClusterManager creates a new oracleClusterManager.ClusterManager
func (o *OKECluster) GetClusterManager() (manager *oracleClusterManager.ClusterManager, err error) {

	oci, err := o.GetOCI()
	if err != nil {
		return manager, err
	}

	return oracleClusterManager.NewClusterManager(oci), nil
}

// GetOCI creates a new oci.OCI
func (o *OKECluster) GetOCI() (OCI *oci.OCI, err error) {

	s, err := o.CommonClusterBase.getSecret(o)
	if err != nil {
		return OCI, err
	}

	OCI, err = oci.NewOCI(secretOracle.CreateOCICredential(s.Values))
	if err != nil {
		return OCI, err
	}

	OCI.SetLogger(log)

	return OCI, err
}

// ListNodeNames returns node names to label them
func (o *OKECluster) ListNodeNames() (pkgCommon.NodeNames, error) {
	// todo implement
	return pkgCommon.NodeNames{}, nil
}
