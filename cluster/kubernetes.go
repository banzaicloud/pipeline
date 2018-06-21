package cluster

import (
	"encoding/base64"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"gopkg.in/yaml.v2"
)

// CreateKubernetesClusterFromRequest creates ClusterModel struct from the request
func CreateKubernetesClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint) (*KubeCluster, error) {

	log.Debug("Create ClusterModel struct from the request")
	var cluster KubeCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Kubernetes: model.KubernetesClusterModel{
			Metadata: request.Properties.CreateKubernetes.Metadata,
		},
	}
	return &cluster, nil

}

// KubeCluster struct for Build your own cluster
type KubeCluster struct {
	modelCluster *model.ClusterModel
	k8sConfig    []byte
	APIEndpoint  string
	CommonClusterBase
}

// CreateCluster creates a new cluster
func (b *KubeCluster) CreateCluster() error {

	// check secret type
	_, err := b.GetSecretWithValidation()
	if err != nil {
		return err
	}

	return nil
}

// Persist save the cluster model
func (b *KubeCluster) Persist(status, statusMessage string) error {
	return b.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (b *KubeCluster) DownloadK8sConfig() ([]byte, error) {
	s, err := b.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	b.k8sConfig, err = base64.StdEncoding.DecodeString(s.GetValue(pkgSecret.K8SConfig))
	return b.k8sConfig, err
}

// GetName returns the name of the cluster
func (b *KubeCluster) GetName() string {
	return b.modelCluster.Name
}

// GetType returns the cloud type of the cluster
func (b *KubeCluster) GetType() string {
	return pkgCluster.Kubernetes
}

// GetStatus gets cluster status
func (b *KubeCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	if len(b.modelCluster.Location) == 0 {
		log.Debug("Empty location.. reload from db")
		// reload from db
		db := model.GetDB()
		db.Find(&b.modelCluster, model.ClusterModel{ID: b.GetID()})
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:        b.modelCluster.Status,
		StatusMessage: b.modelCluster.StatusMessage,
		Name:          b.GetName(),
		Location:      b.modelCluster.Location,
		Cloud:         pkgCluster.Kubernetes,
		ResourceID:    b.modelCluster.ID,
		NodePools:     nil,
	}, nil
}

// DeleteCluster deletes cluster from cloud, in this case no delete function
func (b *KubeCluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates cluster in cloud, in this case no update function
func (b *KubeCluster) UpdateCluster(*pkgCluster.UpdateClusterRequest) error {
	return nil
}

// GetID returns the specified cluster id
func (b *KubeCluster) GetID() uint {
	return b.modelCluster.ID
}

// GetSecretId returns the specified secret id
func (b *KubeCluster) GetSecretId() string {
	return b.modelCluster.SecretId
}

// GetSshSecretId returns the specified ssh secret id
func (b *KubeCluster) GetSshSecretId() string {
	return b.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (b *KubeCluster) SaveSshSecretId(sshSecretId string) error {
	return b.modelCluster.UpdateSshSecret(sshSecretId)
}

// GetModel returns the whole clusterModel
func (b *KubeCluster) GetModel() *model.ClusterModel {
	return b.modelCluster
}

// CheckEqualityToUpdate validates the update request, in this case no update function
func (b *KubeCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return nil
}

// AddDefaultsToUpdate adds defaults to update request, in this case no update function
func (b *KubeCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {

}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (b *KubeCluster) GetAPIEndpoint() (string, error) {

	if b.APIEndpoint != "" {
		return b.APIEndpoint, nil
	}
	secretItem, err := b.GetSecretWithValidation()
	if err != nil {
		return "", err
	}
	config, err := base64.StdEncoding.DecodeString(secretItem.GetValue(pkgSecret.K8SConfig))
	if err != nil {
		return "", err
	}
	kubeConf := kubeConfig{}
	err = yaml.Unmarshal(config, &kubeConf)
	if err != nil {
		return "", err
	}
	b.APIEndpoint = kubeConf.Clusters[0].Cluster.Server
	return b.APIEndpoint, nil
}

// DeleteFromDatabase deletes model from the database
func (b *KubeCluster) DeleteFromDatabase() error {
	return b.modelCluster.Delete()
}

// GetOrganizationId returns the specified organization id
func (b *KubeCluster) GetOrganizationId() uint {
	return b.modelCluster.OrganizationId
}

// CreateKubernetesClusterFromModel converts ClusterModel to KubeCluster
func CreateKubernetesClusterFromModel(clusterModel *model.ClusterModel) (*KubeCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	kubeCluster := KubeCluster{
		modelCluster: clusterModel,
	}
	return &kubeCluster, nil
}

// UpdateStatus updates cluster status in database
func (b *KubeCluster) UpdateStatus(status, statusMessage string) error {
	return b.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (b *KubeCluster) GetClusterDetails() (*pkgCluster.ClusterDetailsResponse, error) {
	status, err := b.GetStatus()
	if err != nil {
		return nil, err
	}

	return &pkgCluster.ClusterDetailsResponse{
		Name: status.Name,
		Id:   status.ResourceID,
	}, nil
}

// ValidateCreationFields validates all field
func (b *KubeCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	return nil
}

// GetSecretWithValidation returns secret from vault
func (b *KubeCluster) GetSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return b.CommonClusterBase.getSecret(b)
}

// GetSshSecretWithValidation returns ssh secret from vault
func (b *KubeCluster) GetSshSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return b.CommonClusterBase.getSecret(b)
}

// SaveConfigSecretId saves the config secret id in database
func (b *KubeCluster) SaveConfigSecretId(configSecretId string) error {
	return b.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (b *KubeCluster) GetConfigSecretId() string {
	return b.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (b *KubeCluster) GetK8sConfig() ([]byte, error) {
	return b.DownloadK8sConfig()
}

// ReloadFromDatabase load cluster from DB
func (b *KubeCluster) ReloadFromDatabase() error {
	return b.modelCluster.ReloadFromDatabase()
}
