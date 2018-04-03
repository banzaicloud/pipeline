package cluster

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"net/http"
)

// CreateBYOCClusterFromRequest creates ClusterModel struct from the request
func CreateBYOCClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*BYOCluster, error) {

	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster BYOCluster

	cluster.modelCluster = &model.ClusterModel{
		Model:            gorm.Model{},
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		OrganizationId:   orgId,
		SecretId:         request.SecretId,
		BYOC: model.BYOClusterModel{
			Metadata: request.Properties.CreateBYOC.Metadata,
		},
	}
	return &cluster, nil

}

// BYOCluster struct for Build your own cluster
type BYOCluster struct {
	modelCluster *model.ClusterModel
	k8sConfig    []byte
	APIEndpoint  string
}

// CreateCluster creates a new cluster
func (b *BYOCluster) CreateCluster() error {

	clusterSecret, err := GetSecret(b)
	if err != nil {
		return err
	}

	if clusterSecret.SecretType != secret.Kubernetes {
		return errors.Errorf("missmatch secret type %s versus %s", clusterSecret.SecretType, secret.Kubernetes)
	}

	return nil
}

// Persist save the cluster model
func (b *BYOCluster) Persist() error {
	return b.modelCluster.Save()
}

// GetK8sConfig returns the Kubernetes config
func (b *BYOCluster) GetK8sConfig() ([]byte, error) {
	var err error = nil
	b.k8sConfig, err = b.createConfig()
	return b.k8sConfig, err
}

// createConfig returns the kubeconfig
func (b *BYOCluster) createConfig() ([]byte, error) {
	if s, err := GetSecret(b); err != nil {
		return nil, err
	} else {
		kubeCluster := kubernetesCluster{
			Name:                  s.GetValue(secret.Name),
			ServiceAccountToken:   s.GetValue(secret.ServiceAccountToken),
			Endpoint:              s.GetValue(secret.Endpoint),
			Username:              s.GetValue(secret.UserName),
			Password:              s.GetValue(secret.Password),
			RootCACert:            s.GetValue(secret.RootCertificate),
			ClientCertificate:     s.GetValue(secret.ClientCertificate),
			ClientKey:             s.GetValue(secret.ClientKey),
			Metadata:              b.modelCluster.BYOC.Metadata,
			AuthProviderName:      s.GetValue(secret.AuthProviderName),
			AuthAccessToken:       s.GetValue(secret.AuthAccessToken),
			AuthAccessTokenExpiry: s.GetValue(secret.AuthAccessTokenExpiry),
			CurrentContext:        s.GetValue(secret.CurrentContext),
		}
		return storeConfig(&kubeCluster, b.GetName())
	}
}

// GetName returns the name of the cluster
func (b *BYOCluster) GetName() string {
	return b.modelCluster.Name
}

// GetType returns the cloud type of the cluster
func (b *BYOCluster) GetType() string {
	return constants.BYOC
}

// GetStatus gets cluster status
func (b *BYOCluster) GetStatus() (*components.GetClusterStatusResponse, error) {

	if len(b.modelCluster.Location) == 0 || len(b.modelCluster.NodeInstanceType) == 0 {
		log.Debug("Empty location and/or nodeInstanceType.. reload from db")
		// reload from db
		db := model.GetDB()
		db.Find(&b.modelCluster, model.ClusterModel{Model: gorm.Model{ID: b.GetID()}})
	}

	return &components.GetClusterStatusResponse{
		Status:           http.StatusOK,
		Name:             b.GetName(),
		Location:         b.modelCluster.Location,
		Cloud:            constants.BYOC,
		NodeInstanceType: b.modelCluster.NodeInstanceType,
		ResourceID:       b.modelCluster.ID,
	}, nil
}

// DeleteCluster deletes cluster from cloud, in this case no delete function
func (b *BYOCluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates cluster in cloud, in this case no update function
func (b *BYOCluster) UpdateCluster(*components.UpdateClusterRequest) error {
	return nil
}

// GetID returns the specified cluster id
func (b *BYOCluster) GetID() uint {
	return b.modelCluster.ID
}

// GetSecretID returns the specified secret id
func (b *BYOCluster) GetSecretID() string {
	return b.modelCluster.SecretId
}

// GetModel returns the whole clusterModel
func (b *BYOCluster) GetModel() *model.ClusterModel {
	return b.modelCluster
}

// CheckEqualityToUpdate validates the update request, in this case no update function
func (b *BYOCluster) CheckEqualityToUpdate(*components.UpdateClusterRequest) error {
	return nil
}

// AddDefaultsToUpdate adds defaults to update request, in this case no update function
func (b *BYOCluster) AddDefaultsToUpdate(*components.UpdateClusterRequest) {

}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (b *BYOCluster) GetAPIEndpoint() (string, error) {

	if b.APIEndpoint != "" {
		return b.APIEndpoint, nil
	}

	if secretItem, err := GetSecret(b); err != nil {
		return "", err
	} else {
		endpoint := secretItem.GetValue(secret.Endpoint)
		b.APIEndpoint = endpoint
		return b.APIEndpoint, nil
	}

}

// DeleteFromDatabase deletes model from the database
func (b *BYOCluster) DeleteFromDatabase() error {
	return b.modelCluster.Delete()
}

// GetOrg returns the specified organization id
func (b *BYOCluster) GetOrg() uint {
	return b.modelCluster.OrganizationId
}

// CreateBYOCClusterFromModel converts ClusterModel to BYOCluster
func CreateBYOCClusterFromModel(clusterModel *model.ClusterModel) (*BYOCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	byocCluster := BYOCluster{
		modelCluster: clusterModel,
	}
	return &byocCluster, nil
}
