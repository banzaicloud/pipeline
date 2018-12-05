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
	"github.com/aws/aws-sdk-go/service/ec2"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cluster"
	banzaicloudDB "github.com/banzaicloud/pipeline/internal/providers/banzaicloud"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/banzaicloud"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var _ CommonCluster = (*EC2ClusterBanzaiCloudDistribution)(nil)

type EC2ClusterBanzaiCloudDistribution struct {
	db            *gorm.DB
	model         *banzaicloudDB.EC2BanzaiCloudClusterModel
	amazonCluster *ec2.EC2 //Don't use this directly
	APIEndpoint   string
	CommonClusterBase
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

func (c *EC2ClusterBanzaiCloudDistribution) SaveSshSecretId(string) error {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) SaveConfigSecretId(string) error {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) GetConfigSecretId() string {
	return c.model.Cluster.ConfigSecretID
}

func (c *EC2ClusterBanzaiCloudDistribution) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	panic("implement me")
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

func (c *EC2ClusterBanzaiCloudDistribution) DeleteFromDatabase() error {
	panic("implement me")
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
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) DownloadK8sConfig() ([]byte, error) {
	// TODO(Ecsy): implement me
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) GetAPIEndpoint() (string, error) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) GetK8sConfig() ([]byte, error) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) RbacEnabled() bool {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) NeedAdminRights() bool {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) GetKubernetesUserName() (string, error) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) ListNodeNames() (common.NodeNames, error) {
	panic("implement me")
}

func (c *EC2ClusterBanzaiCloudDistribution) NodePoolExists(nodePoolName string) bool {
	panic("implement me")
}

func CreateEC2ClusterBanzaiCloudDistributionFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EC2ClusterBanzaiCloudDistribution, error) {
	log.Debug("Create ClusterModel struct from the request")
	c := &EC2ClusterBanzaiCloudDistribution{}

	c.db = pipConfig.DB()
	network, err := createEC2BanzaiCloudNetworkFromRequest(request.Properties.CreateClusterBanzaiCloud.Network, userId)
	if err != nil {
		return nil, err
	}

	nodepools, err := createEC2BanzaiCloudNodePoolsFromRequest(request.Properties.CreateClusterBanzaiCloud.NodePools, userId)
	if err != nil {
		return nil, err
	}

	kubernetes, err := createEC2BanzaiCloudKubernetesFromRequest(request.Properties.CreateClusterBanzaiCloud.Kubernetes, userId)
	if err != nil {
		return nil, err
	}

	kubeADM, err := createEC2BanzaiCloudKubeADMFromRequest(request.Properties.CreateClusterBanzaiCloud.KubeADM, userId)
	if err != nil {
		return nil, err
	}

	cri, err := createEC2BanzaiCloudCRIFromRequest(request.Properties.CreateClusterBanzaiCloud.CRI, userId)
	if err != nil {
		return nil, err
	}

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

func createEC2BanzaiCloudNodePoolsFromRequest(pools banzaicloud.NodePools, userId uint) (banzaicloudDB.NodePools, error) {
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
	return nps, nil
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

func createEC2BanzaiCloudNetworkFromRequest(network banzaicloud.Network, userId uint) (banzaicloudDB.Network, error) {
	n := banzaicloudDB.Network{
		ServiceCIDR:      network.ServiceCIDR,
		PodCIDR:          network.PodCIDR,
		Provider:         convertNetworkProvider(network.Provider),
		APIServerAddress: network.APIServerAddress,
	}
	n.CreatedBy = userId
	return n, nil
}

func convertNetworkProvider(provider banzaicloud.NetworkProvider) (result banzaicloudDB.NetworkProvider) {
	return banzaicloudDB.NetworkProvider(provider)
}

func createEC2BanzaiCloudKubernetesFromRequest(kubernetes banzaicloud.Kubernetes, userId uint) (banzaicloudDB.Kubernetes, error) {
	k := banzaicloudDB.Kubernetes{
		Version: kubernetes.Version,
		RBAC:    banzaicloudDB.RBAC{Enabled: kubernetes.RBAC.Enabled},
	}
	k.CreatedBy = userId
	return k, nil
}

func createEC2BanzaiCloudKubeADMFromRequest(kubernetes banzaicloud.KubeADM, userId uint) (banzaicloudDB.KubeADM, error) {
	a := banzaicloudDB.KubeADM{
		ExtraArgs: convertExtraArgs(kubernetes.ExtraArgs),
	}
	a.CreatedBy = userId
	return a, nil
}

func convertExtraArgs(extraArgs banzaicloud.ExtraArgs) banzaicloudDB.ExtraArgs {
	res := make(banzaicloudDB.ExtraArgs, len(extraArgs))
	for k, v := range extraArgs {
		res[k] = banzaicloudDB.ExtraArg(v)
	}
	return res
}

func createEC2BanzaiCloudCRIFromRequest(cri banzaicloud.CRI, userId uint) (banzaicloudDB.CRI, error) {
	c := banzaicloudDB.CRI{
		Runtime:       banzaicloudDB.Runtime(cri.Runtime),
		RuntimeConfig: cri.RuntimeConfig,
	}
	c.CreatedBy = userId
	return c, nil
}

func getMasterInstanceTypeAndImageFromNodePools(nodepools banzaicloudDB.NodePools) (masterInstanceType string, masterImage string, err error) {
	for _, nodepool := range nodepools {
		for _, role := range nodepool.Roles {
			if banzaicloudDB.RoleMaster == role {
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
