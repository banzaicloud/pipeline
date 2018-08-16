package cluster

import (
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
	"gopkg.in/yaml.v2"
)

// DummyCluster struct for DC
type DummyCluster struct {
	modelCluster *model.ClusterModel
	APIEndpoint  string
}

// CreateDummyClusterFromRequest creates ClusterModel struct from the request
func CreateDummyClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*DummyCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster DummyCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		CreatedBy:      userId,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.Dummy,
		Dummy: model.DummyClusterModel{
			KubernetesVersion: request.Properties.CreateClusterDummy.Node.KubernetesVersion,
			NodeCount:         request.Properties.CreateClusterDummy.Node.Count,
		},
	}
	return &cluster, nil
}

//CreateCluster creates a new cluster
func (d *DummyCluster) CreateCluster() error {
	return nil
}

//Persist save the cluster model
func (d *DummyCluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", d.modelCluster)
	return d.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (d *DummyCluster) DownloadK8sConfig() ([]byte, error) {
	return yaml.Marshal(createDummyConfig())
}

//GetName returns the name of the cluster
func (d *DummyCluster) GetName() string {
	return d.modelCluster.Name
}

//GetCloud returns the cloud type of the cluster
func (d *DummyCluster) GetCloud() string {
	return pkgCluster.Dummy
}

// GetDistribution returns the distribution type of the cluster
func (d *DummyCluster) GetDistribution() string {
	return d.modelCluster.Distribution
}

//GetStatus gets cluster status
func (d *DummyCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	return &pkgCluster.GetClusterStatusResponse{
		Status:            d.modelCluster.Status,
		StatusMessage:     d.modelCluster.StatusMessage,
		Name:              d.modelCluster.Name,
		Location:          d.modelCluster.Location,
		Cloud:             pkgCluster.Dummy,
		Distribution:      pkgCluster.Dummy,
		ResourceID:        d.GetID(),
		CreatorBaseFields: *NewCreatorBaseFields(d.modelCluster.CreatedAt, d.modelCluster.CreatedBy),
		NodePools:         nil,
	}, nil
}

// DeleteCluster deletes cluster
func (d *DummyCluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates the dummy cluster
func (d *DummyCluster) UpdateCluster(r *pkgCluster.UpdateClusterRequest, _ uint) error {
	d.modelCluster.Dummy.KubernetesVersion = r.Dummy.Node.KubernetesVersion
	d.modelCluster.Dummy.NodeCount = r.Dummy.Node.Count
	return nil
}

//GetID returns the specified cluster id
func (d *DummyCluster) GetID() uint {
	return d.modelCluster.ID
}

func (d *DummyCluster) GetUID() string {
	return d.modelCluster.UID
}

//GetModel returns the whole clusterModel
func (d *DummyCluster) GetModel() *model.ClusterModel {
	return d.modelCluster
}

//CheckEqualityToUpdate validates the update request
func (d *DummyCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	return nil
}

//AddDefaultsToUpdate adds defaults to update request
func (d *DummyCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (d *DummyCluster) GetAPIEndpoint() (string, error) {
	d.APIEndpoint = "http://cow.org:8080"
	return d.APIEndpoint, nil
}

//DeleteFromDatabase deletes model from the database
func (d *DummyCluster) DeleteFromDatabase() error {
	return d.modelCluster.Delete()
}

// GetOrganizationId gets org where the cluster belongs
func (d *DummyCluster) GetOrganizationId() uint {
	return d.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (d *DummyCluster) GetLocation() string {
	return d.modelCluster.Location
}

//GetSecretId retrieves the secret id
func (d *DummyCluster) GetSecretId() string {
	return d.modelCluster.SecretId
}

//GetSshSecretId retrieves the ssh secret id
func (d *DummyCluster) GetSshSecretId() string {
	return d.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (d *DummyCluster) SaveSshSecretId(sshSecretId string) error {
	d.modelCluster.SshSecretId = sshSecretId
	return nil
}

// RequiresSshPublicKey returns false
func (d *DummyCluster) RequiresSshPublicKey() bool {
	return true
}

// createDummyConfig creates a (dummy) kubeconfig
func createDummyConfig() *kubeConfig {
	return &kubeConfig{
		APIVersion: "v1",
		Clusters: []configCluster{
			{
				Cluster: dataCluster{
					Server: "http://cow.org:8080",
				},
				Name: "cow-cluster",
			}, {
				Cluster: dataCluster{
					Server: "https://horse.org:4443",
				},
				Name: "horse-cluster",
			},
			{
				Cluster: dataCluster{
					Server: "https://pig.org:443",
				},
				Name: "pig-cluster",
			},
		},
		Contexts: []configContext{
			{
				Context: contextData{
					Cluster: "horse-cluster",
					User:    "green-user",
				},
				Name: "federal-context",
			}, {
				Context: contextData{
					Cluster: "pig-cluster",
					User:    "black-user",
				},
				Name: "queen-anne-context",
			},
		},
		Users: []configUser{
			{
				Name: "blue-user",
				User: userData{
					Token: "blue-token",
				},
			},
			{
				Name: "green-user",
				User: userData{},
			},
		},
		CurrentContext: "federal-context",
		Kind:           "Config",
	}

}

//CreateDummyClusterFromModel creates the cluster from the model
func CreateDummyClusterFromModel(clusterModel *model.ClusterModel) (*DummyCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	dummyCluster := DummyCluster{
		modelCluster: clusterModel,
	}
	return &dummyCluster, nil
}

// UpdateStatus updates cluster status in database
func (d *DummyCluster) UpdateStatus(status, statusMessage string) error {
	return d.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (d *DummyCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	status, err := d.GetStatus()
	if err != nil {
		return nil, err
	}

	return &pkgCluster.DetailsResponse{
		CreatorBaseFields: pkgCommon.CreatorBaseFields{},
		Name:              status.Name,
		Id:                status.ResourceID,
		Location:          status.Location,
		MasterVersion:     "1.9.4",
	}, nil
}

// ValidateCreationFields validates all field
func (d *DummyCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	return nil
}

// GetSecretWithValidation returns secret from vault
func (d *DummyCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return &secret.SecretItemResponse{
		Type: pkgCluster.Dummy,
	}, nil
}

// SaveConfigSecretId saves the config secret id in database
func (d *DummyCluster) SaveConfigSecretId(configSecretId string) error {
	return d.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (d *DummyCluster) GetConfigSecretId() string {
	return d.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (d *DummyCluster) GetK8sConfig() ([]byte, error) {
	return d.DownloadK8sConfig()
}

// ListNodeNames returns node names to label them
func (d *DummyCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	return
}

// RbacEnabled returns true if rbac enabled on the cluster
func (d *DummyCluster) RbacEnabled() bool {
	return d.modelCluster.RbacEnabled
}
