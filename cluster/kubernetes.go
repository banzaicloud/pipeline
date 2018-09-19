// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"encoding/base64"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"gopkg.in/yaml.v2"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateKubernetesClusterFromRequest creates ClusterModel struct from the request
func CreateKubernetesClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*KubeCluster, error) {

	log.Debug("Create ClusterModel struct from the request")
	var cluster KubeCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		CreatedBy:      userId,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.Unknown,
		Kubernetes: model.KubernetesClusterModel{
			Metadata: request.Properties.CreateClusterKubernetes.Metadata,
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
func (c *KubeCluster) CreateCluster() error {

	// check secret type
	_, err := c.GetSecretWithValidation()
	if err != nil {
		return err
	}

	return nil
}

// Persist save the cluster model
func (c *KubeCluster) Persist(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// createDefaultStorageClass creates a default storage class as some clusters are not created with
// any storage classes or with default one
func createDefaultStorageClass(kubernetesClient *kubernetes.Clientset, provisioner string) error {
	defaultStorageClass := storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner: provisioner,
	}

	_, err := kubernetesClient.StorageV1().StorageClasses().Create(&defaultStorageClass)

	return err
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (c *KubeCluster) DownloadK8sConfig() ([]byte, error) {
	s, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	c.k8sConfig, err = base64.StdEncoding.DecodeString(s.GetValue(pkgSecret.K8SConfig))
	return c.k8sConfig, err
}

// GetName returns the name of the cluster
func (c *KubeCluster) GetName() string {
	return c.modelCluster.Name
}

// GetCloud returns the cloud type of the cluster
func (c *KubeCluster) GetCloud() string {
	return pkgCluster.Kubernetes
}

// GetDistribution returns the distribution type of the cluster
func (c *KubeCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

// GetStatus gets cluster status
func (c *KubeCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	if len(c.modelCluster.Location) == 0 {
		log.Debug("Empty location.. reload from db")
		// reload from db
		db := config.DB()
		db.Find(&c.modelCluster, model.ClusterModel{ID: c.GetID()})
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.modelCluster.Status,
		StatusMessage:     c.modelCluster.StatusMessage,
		Name:              c.GetName(),
		Location:          c.modelCluster.Location,
		Cloud:             pkgCluster.Kubernetes,
		Distribution:      c.modelCluster.Distribution,
		ResourceID:        c.modelCluster.ID,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		NodePools:         nil,
	}, nil
}

// DeleteCluster deletes cluster from cloud, in this case no delete function
func (c *KubeCluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates cluster in cloud, in this case no update function
func (c *KubeCluster) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	return nil
}

// GetID returns the specified cluster id
func (c *KubeCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *KubeCluster) GetUID() string {
	return c.modelCluster.UID
}

func (c *KubeCluster) GetCreatorID() uint {
	return c.modelCluster.CreatedBy
}

// GetSecretId returns the specified secret id
func (c *KubeCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

// GetSshSecretId returns the specified ssh secret id
func (c *KubeCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (c *KubeCluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

// GetModel returns the whole clusterModel
func (c *KubeCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

// CheckEqualityToUpdate validates the update request, in this case no update function
func (c *KubeCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return nil
}

// AddDefaultsToUpdate adds defaults to update request, in this case no update function
func (c *KubeCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {

}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *KubeCluster) GetAPIEndpoint() (string, error) {

	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}
	secretItem, err := c.GetSecretWithValidation()
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
	c.APIEndpoint = kubeConf.Clusters[0].Cluster.Server
	return c.APIEndpoint, nil
}

// DeleteFromDatabase deletes model from the database
func (c *KubeCluster) DeleteFromDatabase() error {
	return c.modelCluster.Delete()
}

// GetOrganizationId returns the specified organization id
func (c *KubeCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (c *KubeCluster) GetLocation() string {
	return c.modelCluster.Location
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
func (c *KubeCluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (c *KubeCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {

	return &pkgCluster.DetailsResponse{
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Name:              c.modelCluster.Name,
		Id:                c.modelCluster.ID,
		Location:          c.modelCluster.Location,
	}, nil
}

// ValidateCreationFields validates all field
func (c *KubeCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	return nil
}

// GetSecretWithValidation returns secret from vault
func (c *KubeCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// SaveConfigSecretId saves the config secret id in database
func (c *KubeCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (c *KubeCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (c *KubeCluster) GetK8sConfig() ([]byte, error) {
	return c.DownloadK8sConfig()
}

// ListNodeNames returns node names to label them
func (c *KubeCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	return
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *KubeCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *KubeCluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *KubeCluster) GetKubernetesUserName() (string, error) {
	return "", nil
}
