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
func (c *DummyCluster) CreateCluster() error {
	return nil
}

//Persist save the cluster model
func (c *DummyCluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", c.modelCluster)
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (c *DummyCluster) DownloadK8sConfig() ([]byte, error) {
	return yaml.Marshal(createDummyConfig())
}

//GetName returns the name of the cluster
func (c *DummyCluster) GetName() string {
	return c.modelCluster.Name
}

//GetCloud returns the cloud type of the cluster
func (c *DummyCluster) GetCloud() string {
	return pkgCluster.Dummy
}

// GetDistribution returns the distribution type of the cluster
func (c *DummyCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

//GetStatus gets cluster status
func (c *DummyCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.modelCluster.Status,
		StatusMessage:     c.modelCluster.StatusMessage,
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		Cloud:             pkgCluster.Dummy,
		Distribution:      pkgCluster.Dummy,
		ResourceID:        c.GetID(),
		Logging:           c.GetLogging(),
		Monitoring:        c.GetMonitoring(),
		SecurityScan:      c.GetSecurityScan(),
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		NodePools:         nil,
		Region:            c.modelCluster.Location,
	}, nil
}

// DeleteCluster deletes cluster
func (c *DummyCluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates the dummy cluster
func (c *DummyCluster) UpdateCluster(r *pkgCluster.UpdateClusterRequest, _ uint) error {
	c.modelCluster.Dummy.KubernetesVersion = r.Dummy.Node.KubernetesVersion
	c.modelCluster.Dummy.NodeCount = r.Dummy.Node.Count
	return nil
}

//GetID returns the specified cluster id
func (c *DummyCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *DummyCluster) GetUID() string {
	return c.modelCluster.UID
}

//GetModel returns the whole clusterModel
func (c *DummyCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

//CheckEqualityToUpdate validates the update request
func (c *DummyCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	return nil
}

//AddDefaultsToUpdate adds defaults to update request
func (c *DummyCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *DummyCluster) GetAPIEndpoint() (string, error) {
	c.APIEndpoint = "http://cow.org:8080"
	return c.APIEndpoint, nil
}

//DeleteFromDatabase deletes model from the database
func (c *DummyCluster) DeleteFromDatabase() error {
	return c.modelCluster.Delete()
}

// GetOrganizationId gets org where the cluster belongs
func (c *DummyCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (c *DummyCluster) GetLocation() string {
	return c.modelCluster.Location
}

//GetSecretId retrieves the secret id
func (c *DummyCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

//GetSshSecretId retrieves the ssh secret id
func (c *DummyCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (c *DummyCluster) SaveSshSecretId(sshSecretId string) error {
	c.modelCluster.SshSecretId = sshSecretId
	return nil
}

// RequiresSshPublicKey returns false
func (c *DummyCluster) RequiresSshPublicKey() bool {
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
	dummyCluster := DummyCluster{
		modelCluster: clusterModel,
	}
	return &dummyCluster, nil
}

// UpdateStatus updates cluster status in database
func (c *DummyCluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// NodePoolExists returns true if node pool with nodePoolName exists
func (c *DummyCluster) NodePoolExists(nodePoolName string) bool {
	return false
}

// GetClusterDetails gets cluster details from cloud
func (c *DummyCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	status, err := c.GetStatus()
	if err != nil {
		return nil, err
	}

	return &pkgCluster.DetailsResponse{
		Id:            status.ResourceID,
		MasterVersion: "1.9.4",
	}, nil
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *DummyCluster) IsReady() (bool, error) {
	return true, nil
}

// ValidateCreationFields validates all field
func (c *DummyCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	return nil
}

// GetSecretWithValidation returns secret from vault
func (c *DummyCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return &secret.SecretItemResponse{
		Type: pkgCluster.Dummy,
	}, nil
}

// SaveConfigSecretId saves the config secret id in database
func (c *DummyCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (c *DummyCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (c *DummyCluster) GetK8sConfig() ([]byte, error) {
	return c.DownloadK8sConfig()
}

// ListNodeNames returns node names to label them
func (c *DummyCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	return
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *DummyCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetSecurityScan returns true if security scan enabled on the cluster
func (c *DummyCluster) GetSecurityScan() bool {
	return c.modelCluster.SecurityScan
}

// SetSecurityScan returns true if security scan enabled on the cluster
func (c *DummyCluster) SetSecurityScan(scan bool) {
	c.modelCluster.SecurityScan = scan
}

// GetLogging returns true if logging enabled on the cluster
func (c *DummyCluster) GetLogging() bool {
	return c.modelCluster.Logging
}

// SetLogging returns true if logging enabled on the cluster
func (c *DummyCluster) SetLogging(l bool) {
	c.modelCluster.Logging = l
}

// GetMonitoring returns true if momnitoring enabled on the cluster
func (c *DummyCluster) GetMonitoring() bool {
	return c.modelCluster.Monitoring
}

// SetMonitoring returns true if monitoring enabled on the cluster
func (c *DummyCluster) SetMonitoring(l bool) {
	c.modelCluster.Monitoring = l
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *DummyCluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *DummyCluster) GetKubernetesUserName() (string, error) {
	return "", nil
}

// GetCreatedBy returns cluster create userID.
func (c *DummyCluster) GetCreatedBy() uint {
	return c.modelCluster.CreatedBy
}
