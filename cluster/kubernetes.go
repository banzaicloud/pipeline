package cluster

import (
	"encoding/base64"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// CreateKubeClusterFromRequest creates ClusterModel struct from the request
func CreateKubernetesClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*Kubecluster, error) {

	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster Kubecluster

	cluster.modelCluster = &model.ClusterModel{
		Model:            gorm.Model{},
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		OrganizationId:   orgId,
		SecretId:         request.SecretId,
		Kubernetes: model.KubernetesClusterModel{
			Metadata: request.Properties.CreateKubernetes.Metadata,
		},
	}
	return &cluster, nil

}

// Kubecluster struct for Build your own cluster
type Kubecluster struct {
	modelCluster *model.ClusterModel
	k8sConfig    []byte
	APIEndpoint  string
}

// CreateCluster creates a new cluster
func (b *Kubecluster) CreateCluster() error {

	clusterSecret, err := GetSecret(b)
	if err != nil {
		return err
	}

	if clusterSecret.SecretType != constants.Kubernetes {
		return errors.Errorf("missmatch secret type %s versus %s", clusterSecret.SecretType, constants.Kubernetes)
	}

	return nil
}

// Persist save the cluster model
func (b *Kubecluster) Persist(status string) error {
	return b.modelCluster.UpdateStatus(status)
}

// GetK8sConfig returns the Kubernetes config
func (b *Kubecluster) GetK8sConfig() ([]byte, error) {
	s, err := GetSecret(b)
	if err != nil {
		return nil, err
	}
	b.k8sConfig, err = base64.StdEncoding.DecodeString(s.GetValue(secret.K8SConfig))
	return b.k8sConfig, err
}

// GetName returns the name of the cluster
func (b *Kubecluster) GetName() string {
	return b.modelCluster.Name
}

// GetType returns the cloud type of the cluster
func (b *Kubecluster) GetType() string {
	return constants.Kubernetes
}

// GetStatus gets cluster status
func (b *Kubecluster) GetStatus() (*components.GetClusterStatusResponse, error) {

	if len(b.modelCluster.Location) == 0 || len(b.modelCluster.NodeInstanceType) == 0 {
		log.Debug("Empty location and/or nodeInstanceType.. reload from db")
		// reload from db
		db := model.GetDB()
		db.Find(&b.modelCluster, model.ClusterModel{Model: gorm.Model{ID: b.GetID()}})
	}

	return &components.GetClusterStatusResponse{
		Status:           b.modelCluster.Status,
		Name:             b.GetName(),
		Location:         b.modelCluster.Location,
		Cloud:            constants.Kubernetes,
		NodeInstanceType: b.modelCluster.NodeInstanceType,
		ResourceID:       b.modelCluster.ID,
	}, nil
}

// DeleteCluster deletes cluster from cloud, in this case no delete function
func (b *Kubecluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates cluster in cloud, in this case no update function
func (b *Kubecluster) UpdateCluster(*components.UpdateClusterRequest) error {
	return nil
}

func (b *Kubecluster) UpdateClusterModelFromRequest(*components.UpdateClusterRequest) {
	// BYOC not supports update cluster
}

// GetID returns the specified cluster id
func (b *Kubecluster) GetID() uint {
	return b.modelCluster.ID
}

// GetSecretID returns the specified secret id
func (b *Kubecluster) GetSecretID() string {
	return b.modelCluster.SecretId
}

// GetModel returns the whole clusterModel
func (b *Kubecluster) GetModel() *model.ClusterModel {
	return b.modelCluster
}

// CheckEqualityToUpdate validates the update request, in this case no update function
func (b *Kubecluster) CheckEqualityToUpdate(*components.UpdateClusterRequest) error {
	return nil
}

// AddDefaultsToUpdate adds defaults to update request, in this case no update function
func (b *Kubecluster) AddDefaultsToUpdate(*components.UpdateClusterRequest) {

}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (b *Kubecluster) GetAPIEndpoint() (string, error) {

	if b.APIEndpoint != "" {
		return b.APIEndpoint, nil
	}
	secretItem, err := GetSecret(b)
	if err != nil {
		return "", err
	}
	config, err := base64.StdEncoding.DecodeString(secretItem.GetValue(secret.K8SConfig))
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
func (b *Kubecluster) DeleteFromDatabase() error {
	return b.modelCluster.Delete()
}

// GetOrg returns the specified organization id
func (b *Kubecluster) GetOrg() uint {
	return b.modelCluster.OrganizationId
}

// CreateBYOCClusterFromModel converts ClusterModel to Kubecluster
func CreateKubernetesClusterFromModel(clusterModel *model.ClusterModel) (*Kubecluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	byocCluster := Kubecluster{
		modelCluster: clusterModel,
	}
	return &byocCluster, nil
}

func (b *Kubecluster) UpdateStatus(status string) error {
	return b.modelCluster.UpdateStatus(status)
}

func (b *Kubecluster) GetClusterDetails() (*components.ClusterDetailsResponse, error) {
	status, err := b.GetStatus()
	if err != nil {
		return nil, err
	}

	return &components.ClusterDetailsResponse{
		Name: status.Name,
		Id:   status.ResourceID,
	}, nil
}

// ValidateCreationFields validates all field
func (b *Kubecluster) ValidateCreationFields(r *components.CreateClusterRequest) error {
	return nil
}
