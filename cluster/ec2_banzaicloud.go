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
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cluster"
	banzaicloudDB "github.com/banzaicloud/pipeline/internal/providers/banzaicloud"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/banzaicloud"
	"github.com/banzaicloud/pipeline/pkg/common"
	pkgError "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var _ CommonCluster = (*EC2ClusterBanzaiCloudDistribution)(nil)

type EC2ClusterBanzaiCloudDistribution struct {
	db    *gorm.DB
	model *banzaicloudDB.EC2BanzaiCloudClusterModel
	//amazonCluster *ec2.EC2 //Don't use this directly
	APIEndpoint string
	CommonClusterBase
}

func (c *EC2ClusterBanzaiCloudDistribution) GetSecurityScan() bool {
	return c.model.Cluster.SecurityScan
}

func (c *EC2ClusterBanzaiCloudDistribution) SetSecurityScan(scan bool) {
	c.model.Cluster.SecurityScan = scan
}

func (c *EC2ClusterBanzaiCloudDistribution) GetLogging() bool {
	return c.model.Cluster.Logging
}

func (c *EC2ClusterBanzaiCloudDistribution) SetLogging(l bool) {
	c.model.Cluster.Logging = l
}

func (c *EC2ClusterBanzaiCloudDistribution) GetMonitoring() bool {
	return c.model.Cluster.Monitoring
}

func (c *EC2ClusterBanzaiCloudDistribution) SetMonitoring(m bool) {
	c.model.Cluster.Monitoring = m
}

func (c *EC2ClusterBanzaiCloudDistribution) GetID() uint {
	return c.model.Cluster.ID
}

func (c *EC2ClusterBanzaiCloudDistribution) GetUID() string {
	return c.model.Cluster.UID
}

func (c *EC2ClusterBanzaiCloudDistribution) GetOrganizationId() uint {
	return c.model.Cluster.OrganizationID
}

func (c *EC2ClusterBanzaiCloudDistribution) GetName() string {
	return c.model.Cluster.Name
}

func (c *EC2ClusterBanzaiCloudDistribution) GetCloud() string {
	return c.model.Cluster.Cloud
}

func (c *EC2ClusterBanzaiCloudDistribution) GetDistribution() string {
	return c.model.Cluster.Distribution
}

func (c *EC2ClusterBanzaiCloudDistribution) GetLocation() string {
	return c.model.Cluster.Location
}

func (c *EC2ClusterBanzaiCloudDistribution) GetCreatedBy() uint {
	return c.model.Cluster.CreatedBy
}

func (c *EC2ClusterBanzaiCloudDistribution) GetSecretId() string {
	return c.model.Cluster.SecretID
}

func (c *EC2ClusterBanzaiCloudDistribution) GetSshSecretId() string {
	return c.model.Cluster.SSHSecretID
}

func (c *EC2ClusterBanzaiCloudDistribution) SaveSshSecretId(sshSecretId string) error {
	c.model.Cluster.SSHSecretID = sshSecretId

	err := c.db.Save(&c.model).Error
	if err != nil {
		return emperror.WrapWith(err, "failed to save ssh secret", "secret", sshSecretId)
	}

	return nil
}

func (c *EC2ClusterBanzaiCloudDistribution) SaveConfigSecretId(configSecretId string) error {
	c.model.Cluster.ConfigSecretID = configSecretId

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.Wrap(err, "failed to save config secret id")
	}

	return nil
}

func (c *EC2ClusterBanzaiCloudDistribution) GetConfigSecretId() string {
	return c.model.Cluster.ConfigSecretID
}

func (c *EC2ClusterBanzaiCloudDistribution) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

func (c *EC2ClusterBanzaiCloudDistribution) Persist(string, string) error {
	err := c.db.Save(c.model).Error
	return err
}

func (c *EC2ClusterBanzaiCloudDistribution) UpdateStatus(status, statusMessage string) error {
	c.model.Cluster.Status = status
	c.model.Cluster.StatusMessage = statusMessage

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.Wrap(err, "failed to update status")
	}
	return nil
}

// DeleteFromDatabase deletes the distribution related entities from the database
func (c *EC2ClusterBanzaiCloudDistribution) DeleteFromDatabase() error {

	// dependencies are deleted using a GORM hook!
	if e := c.db.Delete(c.model).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete EC2BanzaiCloudCluster", "distro", c.model.ID)
	}

	return nil
}

func (c *EC2ClusterBanzaiCloudDistribution) CreateCluster() error {
	return nil
}

func (c *EC2ClusterBanzaiCloudDistribution) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	// TODO(Ecsy): implement me
	return nil
}

func (c *EC2ClusterBanzaiCloudDistribution) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) DeleteCluster() error {
	// do nothing (the cluster should be left on the provider for now
	return nil
}

func (c *EC2ClusterBanzaiCloudDistribution) DownloadK8sConfig() ([]byte, error) {
	return nil, pkgError.ErrorFunctionShouldNotBeCalled
}

func (c *EC2ClusterBanzaiCloudDistribution) GetAPIEndpoint() (string, error) {
	// TODO: endpoint url
	return "", nil
}

func (c *EC2ClusterBanzaiCloudDistribution) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func (c *EC2ClusterBanzaiCloudDistribution) RbacEnabled() bool {
	return c.model.Kubernetes.RBACEnabled
}

func (c *EC2ClusterBanzaiCloudDistribution) NeedAdminRights() bool {
	return false
}

func (c *EC2ClusterBanzaiCloudDistribution) GetKubernetesUserName() (string, error) {
	return "", nil
}

func (c *EC2ClusterBanzaiCloudDistribution) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.model.NodePools {
		providerConfig := banzaicloudDB.NodePoolProviderConfigAmazon{}
		err := mapstructure.Decode(np.ProviderConfig, &providerConfig)
		if err != nil {
			return nil, emperror.WrapWith(err, "failed to decode providerconfig", "cluster", c.model.Cluster.Name)
		}
		nodePools[np.Name] = &pkgCluster.NodePoolStatus{
			Count:        len(np.Hosts),
			InstanceType: providerConfig.AutoScalingGroup.InstanceType,
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.model.Cluster.Status,
		StatusMessage:     c.model.Cluster.StatusMessage,
		Name:              c.model.Cluster.Name,
		Location:          c.model.Cluster.Location,
		Cloud:             c.model.Cluster.Cloud,
		Distribution:      c.model.Cluster.Distribution,
		ResourceID:        c.model.Cluster.ID,
		Logging:           c.GetLogging(),
		Monitoring:        c.GetMonitoring(),
		SecurityScan:      c.GetSecurityScan(),
		NodePools:         nodePools,
		Version:           c.model.Kubernetes.Version,
		CreatorBaseFields: *NewCreatorBaseFields(c.model.Cluster.CreatedAt, c.model.Cluster.CreatedBy),
		Region:            c.model.Cluster.Location,
	}, nil
}

func (c *EC2ClusterBanzaiCloudDistribution) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	log.Info("Getting cluster details")

	status, err := c.GetStatus()
	if err != nil {
		return nil, err
	}

	nodePools := make(map[string]*pkgCluster.NodePoolDetails)
	for _, np := range c.model.NodePools {
		nodePools[np.Name] = &pkgCluster.NodePoolDetails{
			CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
			NodePoolStatus:    *status.NodePools[np.Name],
		}
	}
	//TODO implement cluster running state check if it is ready
	return &pkgCluster.DetailsResponse{
		Id:                       c.model.Cluster.ID,
		MasterVersion:            c.model.Kubernetes.Version,
		NodePools:                nodePools,
		GetClusterStatusResponse: *status,
	}, nil
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *EC2ClusterBanzaiCloudDistribution) IsReady() (bool, error) {
	// TODO: is this a correct implementation?
	return true, nil
}

func (c *EC2ClusterBanzaiCloudDistribution) ListNodeNames() (common.NodeNames, error) {
	var nodes = make(map[string][]string)
	for _, nodepool := range c.model.NodePools {
		for _, host := range nodepool.Hosts {
			nodes[nodepool.Name] = append(nodes[nodepool.Name], host.Name)
		}
	}
	return nodes, nil
}

func (c *EC2ClusterBanzaiCloudDistribution) NodePoolExists(nodePoolName string) bool {
	for _, np := range c.model.NodePools {
		if np.Name == nodePoolName {
			return true
		}
	}
	return false
}

func CreateEC2ClusterBanzaiCloudDistributionFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EC2ClusterBanzaiCloudDistribution, error) {
	log.Debug("Create ClusterModel struct from the request")
	c := &EC2ClusterBanzaiCloudDistribution{}

	c.db = pipConfig.DB()

	var (
		network    = createEC2BanzaiCloudNetworkFromRequest(request.Properties.CreateClusterBanzaiCloud.Network, userId)
		nodepools  = createEC2BanzaiCloudNodePoolsFromRequest(request.Properties.CreateClusterBanzaiCloud.NodePools, userId)
		kubernetes = createEC2BanzaiCloudKubernetesFromRequest(request.Properties.CreateClusterBanzaiCloud.Kubernetes, userId)
		kubeADM    = createEC2BanzaiCloudKubeADMFromRequest(request.Properties.CreateClusterBanzaiCloud.KubeADM, userId)
		cri        = createEC2BanzaiCloudCRIFromRequest(request.Properties.CreateClusterBanzaiCloud.CRI, userId)
	)

	instanceType, image, err := getMasterInstanceTypeAndImageFromNodePools(nodepools)
	if err != nil {
		return nil, err
	}

	c.model = &banzaicloudDB.EC2BanzaiCloudClusterModel{
		Cluster: cluster.ClusterModel{
			Name:           request.Name,
			Location:       request.Location,
			Cloud:          request.Cloud,
			Distribution:   pkgCluster.BanzaiCloud,
			OrganizationID: orgId,
			RbacEnabled:    kubernetes.RBAC.Enabled,
			CreatedBy:      userId,
		},
		MasterInstanceType: instanceType,
		MasterImage:        image,
		Network:            network,
		NodePools:          nodepools,
		Kubernetes:         kubernetes,
		KubeADM:            kubeADM,
		CRI:                cri,
	}

	return c, nil
}

func CreateEC2ClusterBanzaiCloudDistributionFromModel(modelCluster *model.ClusterModel) (*EC2ClusterBanzaiCloudDistribution, error) {
	log.Debug("Create ClusterModel struct from the request")
	db := pipConfig.DB()

	m := banzaicloudDB.EC2BanzaiCloudClusterModel{
		ClusterID: modelCluster.ID,
	}

	log.Debug("Load EC2 props from database")
	err := db.Where(m).
		Preload("Cluster").
		Preload("Network").
		Preload("NodePools").
		Preload("Kubernetes").
		Preload("KubeADM").
		Preload("CRI").
		First(&m).
		Error
	if err != nil {
		return nil, err
	}

	c := &EC2ClusterBanzaiCloudDistribution{
		db:    db,
		model: &m,
	}
	return c, nil
}

func createEC2BanzaiCloudNodePoolsFromRequest(pools banzaicloud.NodePools, userId uint) banzaicloudDB.NodePools {
	var nps banzaicloudDB.NodePools

	for _, pool := range pools {
		np := banzaicloudDB.NodePool{
			Name:           pool.Name,
			Roles:          convertRoles(pool.Roles),
			Hosts:          convertHosts(pool.Hosts),
			Provider:       convertNodePoolProvider(pool.Provider),
			ProviderConfig: pool.ProviderConfig,
		}
		np.CreatedBy = userId
		nps = append(nps, np)
	}
	return nps
}

func convertRoles(roles banzaicloud.Roles) (result banzaicloudDB.Roles) {
	for _, role := range roles {
		result = append(result, banzaicloudDB.Role(role))
	}
	return
}

func convertHosts(hosts banzaicloud.Hosts) (result banzaicloudDB.Hosts) {
	for _, host := range hosts {
		result = append(result, banzaicloudDB.Host{
			Name:             host.Name,
			PrivateIP:        host.PrivateIP,
			NetworkInterface: host.NetworkInterface,
			Roles:            convertRoles(host.Roles),
			Labels:           convertLabels(host.Labels),
			Taints:           convertTaints(host.Taints),
		})
	}

	return
}

func convertNodePoolProvider(provider banzaicloud.NodePoolProvider) (result banzaicloudDB.NodePoolProvider) {
	return banzaicloudDB.NodePoolProvider(provider)
}

func convertLabels(labels banzaicloud.Labels) banzaicloudDB.Labels {
	res := make(banzaicloudDB.Labels, len(labels))
	for k, v := range labels {
		res[k] = v
	}
	return res
}

func convertTaints(taints banzaicloud.Taints) (result banzaicloudDB.Taints) {
	for _, taint := range taints {
		result = append(result, banzaicloudDB.Taint(taint))
	}
	return
}

func createEC2BanzaiCloudNetworkFromRequest(network banzaicloud.Network, userId uint) banzaicloudDB.Network {
	n := banzaicloudDB.Network{
		ServiceCIDR:      network.ServiceCIDR,
		PodCIDR:          network.PodCIDR,
		Provider:         convertNetworkProvider(network.Provider),
		APIServerAddress: network.APIServerAddress,
	}
	n.CreatedBy = userId
	return n
}

func convertNetworkProvider(provider banzaicloud.NetworkProvider) (result banzaicloudDB.NetworkProvider) {
	return banzaicloudDB.NetworkProvider(provider)
}

func createEC2BanzaiCloudKubernetesFromRequest(kubernetes banzaicloud.Kubernetes, userId uint) banzaicloudDB.Kubernetes {
	k := banzaicloudDB.Kubernetes{
		Version: kubernetes.Version,
		RBAC:    banzaicloudDB.RBAC{Enabled: kubernetes.RBAC.Enabled},
	}
	k.CreatedBy = userId
	return k
}

func createEC2BanzaiCloudKubeADMFromRequest(kubernetes banzaicloud.KubeADM, userId uint) banzaicloudDB.KubeADM {
	a := banzaicloudDB.KubeADM{
		ExtraArgs: convertExtraArgs(kubernetes.ExtraArgs),
	}
	a.CreatedBy = userId
	return a
}

func convertExtraArgs(extraArgs banzaicloud.ExtraArgs) banzaicloudDB.ExtraArgs {
	res := make(banzaicloudDB.ExtraArgs, len(extraArgs))
	for k, v := range extraArgs {
		res[k] = banzaicloudDB.ExtraArg(v)
	}
	return res
}

func createEC2BanzaiCloudCRIFromRequest(cri banzaicloud.CRI, userId uint) banzaicloudDB.CRI {
	c := banzaicloudDB.CRI{
		Runtime:       banzaicloudDB.Runtime(cri.Runtime),
		RuntimeConfig: cri.RuntimeConfig,
	}
	c.CreatedBy = userId
	return c
}

func getMasterInstanceTypeAndImageFromNodePools(nodepools banzaicloudDB.NodePools) (masterInstanceType string, masterImage string, err error) {
	for _, nodepool := range nodepools {
		for _, role := range nodepool.Roles {
			if role == banzaicloudDB.RoleMaster {
				switch nodepool.Provider {
				case banzaicloudDB.NPPAmazon:
					providerConfig := banzaicloudDB.NodePoolProviderConfigAmazon{}
					err = mapstructure.Decode(nodepool.ProviderConfig, &providerConfig)
					if err != nil {
						return
					}
					masterInstanceType = providerConfig.AutoScalingGroup.InstanceType
					masterImage = providerConfig.AutoScalingGroup.Image
					return
				}
			}
		}
	}
	return
}
