package cluster

import (
	"fmt"

	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
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

// CreateClusterFromRequest creates ClusterModel struct from the request
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
func (d *OKECluster) CreateCluster() error {

	log.Info("Start creating Oracle cluster")

	cm, err := d.GetClusterManager()
	if err != nil {
		return err
	}

	return cm.ManageOKECluster(&d.modelCluster.Oracle)
}

// UpdateCluster updates the cluster
func (d *OKECluster) UpdateCluster(r *pkgCluster.UpdateClusterRequest) error {

	model, err := modelOracle.CreateModelFromUpdateRequest(d.modelCluster.Oracle, r)
	if err != nil {
		return err
	}

	cm, err := d.GetClusterManager()
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
	d.modelCluster.Oracle = model

	return err
}

// DeleteCluster deletes cluster
func (d *OKECluster) DeleteCluster() error {

	// mark cluster model to deleting
	d.modelCluster.Oracle.Delete = true

	cm, err := d.GetClusterManager()
	if err != nil {
		return err
	}

	return cm.ManageOKECluster(&d.modelCluster.Oracle)
}

//Persist save the cluster model
func (d *OKECluster) Persist(status, statusMessage string) error {

	return d.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (d *OKECluster) DownloadK8sConfig() ([]byte, error) {

	oci, err := d.GetOCI()
	if err != nil {
		return nil, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	return ce.GetK8SConfig(d.modelCluster.Oracle.OCID)
}

//GetName returns the name of the cluster
func (d *OKECluster) GetName() string {
	return d.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (d *OKECluster) GetType() string {
	return pkgCluster.Oracle
}

//GetStatus gets cluster status
func (d *OKECluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	return &pkgCluster.GetClusterStatusResponse{
		Status:        d.modelCluster.Status,
		StatusMessage: d.modelCluster.StatusMessage,
		Name:          d.modelCluster.Name,
		Location:      d.modelCluster.Location,
		Cloud:         pkgCluster.Oracle,
		ResourceID:    d.GetID(),
		NodePools:     nil,
	}, nil
}

//GetID returns the specified cluster id
func (d *OKECluster) GetID() uint {
	return d.modelCluster.ID
}

//GetModel returns the whole clusterModel
func (d *OKECluster) GetModel() *model.ClusterModel {
	return d.modelCluster
}

//CheckEqualityToUpdate validates the update request
func (d *OKECluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	return nil
}

//AddDefaultsToUpdate adds defaults to update request
func (d *OKECluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (d *OKECluster) GetAPIEndpoint() (string, error) {

	oci, err := d.GetOCI()
	if err != nil {
		return d.APIEndpoint, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return d.APIEndpoint, err
	}

	cluster, err := ce.GetCluster(d.modelCluster.Oracle.OCID)
	if err != nil {
		return d.APIEndpoint, err
	}

	d.APIEndpoint = fmt.Sprintf("https://%s", *cluster.Endpoints.Kubernetes)

	return d.APIEndpoint, nil
}

// DeleteFromDatabase deletes model from the database
func (g *OKECluster) DeleteFromDatabase() error {
	err := g.modelCluster.Delete()
	if err != nil {
		return err
	}

	err = g.modelCluster.Oracle.Cleanup()
	if err != nil {
		return err
	}

	g.modelCluster = nil
	return nil
}

// GetOrganizationId gets org where the cluster belongs
func (d *OKECluster) GetOrganizationId() uint {
	return d.modelCluster.OrganizationId
}

//GetSecretId retrieves the secret id
func (d *OKECluster) GetSecretId() string {
	return d.modelCluster.SecretId
}

//GetSshSecretId retrieves the ssh secret id
func (d *OKECluster) GetSshSecretId() string {
	return d.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (g *OKECluster) SaveSshSecretId(sshSecretId string) error {
	return g.modelCluster.UpdateSshSecret(sshSecretId)
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
func (d *OKECluster) UpdateStatus(status, statusMessage string) error {
	return d.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (d *OKECluster) GetClusterDetails() (*pkgCluster.ClusterDetailsResponse, error) {
	status, err := d.GetStatus()
	if err != nil {
		return nil, err
	}

	return &pkgCluster.ClusterDetailsResponse{
		Name: status.Name,
		Id:   status.ResourceID,
	}, nil
}

// ValidateCreationFields validates all field
func (d *OKECluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

	cm, err := d.GetClusterManager()
	if err != nil {
		return err
	}

	return cm.ValidateModel(&d.modelCluster.Oracle)
}

// GetSecretWithValidation returns secret from vault
func (g *OKECluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return g.CommonClusterBase.getSecret(g)
}

// GetSshSecretWithValidation returns ssh secret from vault
func (g *OKECluster) GetSshSecretWithValidation() (*secret.SecretItemResponse, error) {
	return g.CommonClusterBase.getSecret(g)
}

// SaveConfigSecretId saves the config secret id in database
func (g *OKECluster) SaveConfigSecretId(configSecretId string) error {
	return g.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (d *OKECluster) GetConfigSecretId() string {
	return d.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (d *OKECluster) GetK8sConfig() ([]byte, error) {
	return d.DownloadK8sConfig()
}

// ReloadFromDatabase load cluster from DB
func (d *OKECluster) ReloadFromDatabase() error {
	return d.modelCluster.ReloadFromDatabase()
}

// GetClusterManager creates a new oracleClusterManager.ClusterManager
func (d *OKECluster) GetClusterManager() (manager *oracleClusterManager.ClusterManager, err error) {

	oci, err := d.GetOCI()
	if err != nil {
		return manager, err
	}

	return oracleClusterManager.NewClusterManager(oci), nil
}

// GetOCI creates a new oci.OCI
func (d *OKECluster) GetOCI() (OCI *oci.OCI, err error) {

	s, err := d.CommonClusterBase.getSecret(d)
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
