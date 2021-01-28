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
	"strings"

	"emperror.dev/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

const RBAC_API_VERSION = "rbac.authorization.k8s.io"

// CreateKubernetesClusterFromRequest creates ClusterModel struct from the request
func CreateKubernetesClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*KubeCluster, error) {
	cluster := KubeCluster{
		log: log.WithField("cluster", request.Name),
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		CreatedBy:      userId,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.Unknown,
		Kubernetes: kubernetesadapter.KubernetesClusterModel{
			Metadata: request.Properties.CreateClusterKubernetes.Metadata,
		},
	}
	updateScaleOptions(&cluster.modelCluster.ScaleOptions, request.ScaleOptions)
	return &cluster, nil
}

// KubeCluster struct for Build your own cluster
type KubeCluster struct {
	modelCluster *model.ClusterModel
	k8sConfig    []byte
	APIEndpoint  string
	log          logrus.FieldLogger

	CommonClusterBase
}

// CreateCluster creates a new cluster
func (c *KubeCluster) CreateCluster() error {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "couldn't get Kubernetes config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "couldn't create Kubernetes client")
	}

	c.modelCluster.RbacEnabled, err = c.isRBACEnabled(client)
	if err != nil {
		return errors.WrapIf(err, "couldn't determine if RBAC is enabled on the cluster")
	}

	if c.modelCluster.RbacEnabled {
		c.log.Info("rbac is enabled on the cluster")
	} else {
		c.log.Info("rbac is not enabled on the cluster")
	}

	return nil
}

// Persist save the cluster model
// Deprecated: Do not use.
func (c *KubeCluster) Persist() error {
	return errors.WrapIf(c.modelCluster.Save(), "failed to persist cluster")
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (c *KubeCluster) DownloadK8sConfig() ([]byte, error) {
	s, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	c.k8sConfig, err = base64.StdEncoding.DecodeString(s.Values[secrettype.K8SConfig])
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
		c.log.Debug("Empty location.. reload from db")
		// reload from db
		db := global.DB()
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
		Region:            c.modelCluster.Location,
		StartedAt:         c.modelCluster.StartedAt,
	}, nil
}

// DeleteCluster deletes cluster from cloud, in this case no delete function
func (c *KubeCluster) DeleteCluster() error {
	return nil
}

// UpdateNodePools updates nodes pools of a cluster
func (c *KubeCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	return nil
}

// UpdateCluster updates cluster in cloud, in this case no update function
func (c *KubeCluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest, _ uint) error {
	return nil
}

// GetID returns the specified cluster id
func (c *KubeCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *KubeCluster) GetUID() string {
	return c.modelCluster.UID
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
	config, err := base64.StdEncoding.DecodeString(secretItem.Values[secrettype.K8SConfig])
	if err != nil {
		return "", err
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
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
	kubeCluster := KubeCluster{
		modelCluster: clusterModel,
	}
	return &kubeCluster, nil
}

// SetStatus sets the cluster's status
func (c *KubeCluster) SetStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// NodePoolExists returns true if node pool with nodePoolName exists
func (c *KubeCluster) NodePoolExists(nodePoolName string) bool {
	return false
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *KubeCluster) IsReady() (bool, error) {
	return true, nil
}

// ValidateCreationFields validates all field
func (c *KubeCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	sir, err := getSecret(c.GetOrganizationId(), r.SecretId)
	if err != nil {
		return errors.WrapIfWithDetails(err, "secret not found", "secretID", r.SecretId)
	}

	return verify.CreateKubeConfigSecretVerifier(sir.Values).VerifySecret()
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

// GetK8sUserConfig returns the Kubernetes config
func (c *KubeCluster) GetK8sUserConfig() ([]byte, error) {
	return c.GetK8sConfig()
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *KubeCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// getScaleOptionsFromModelV1 returns scale options for the cluster
func (c *KubeCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.modelCluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *KubeCluster) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.modelCluster.ScaleOptions, scaleOptions)
}

// isRBACEnabled determines if RBAC is enabled on the Kubernetes cluster by investigating if list of
// api versions enabled on the API server contains 'rbac`
func (c *KubeCluster) isRBACEnabled(client *kubernetes.Clientset) (bool, error) {
	apiGroups, err := client.ServerGroups()
	if err != nil {
		return false, errors.WrapIf(err, "couldn't retrieve Kubernetes API groups")
	}

	if apiGroups == nil {
		return false, errors.New("no API groups found")
	}

	for _, g := range apiGroups.Groups {
		if strings.Contains(strings.ToLower(g.Name), RBAC_API_VERSION) {
			return true, nil
		}
	}

	return false, nil
}
