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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/ack"
	"github.com/banzaicloud/pipeline/pkg/cluster/ack/action"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/goph/emperror"
	jmespath "github.com/jmespath/go-jmespath"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes"
)

var _ CommonCluster = (*ACKCluster)(nil)

type ACKCluster struct {
	alibabaCluster *ack.AlibabaDescribeClusterResponse
	modelCluster   *model.ClusterModel
	APIEndpoint    string
	log            logrus.FieldLogger
	CommonClusterBase
}

func (c *ACKCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetSecurityScan returns true if security scan enabled on the cluster
func (c *ACKCluster) GetSecurityScan() bool {
	return c.modelCluster.SecurityScan
}

// SetSecurityScan returns true if security scan enabled on the cluster
func (c *ACKCluster) SetSecurityScan(scan bool) {
	c.modelCluster.SecurityScan = scan
}

// GetLogging returns true if logging enabled on the cluster
func (c *ACKCluster) GetLogging() bool {
	return c.modelCluster.Logging
}

// SetLogging returns true if logging enabled on the cluster
func (c *ACKCluster) SetLogging(l bool) {
	c.modelCluster.Logging = l
}

// GetMonitoring returns true if momnitoring enabled on the cluster
func (c *ACKCluster) GetMonitoring() bool {
	return c.modelCluster.Monitoring
}

// SetMonitoring returns true if monitoring enabled on the cluster
func (c *ACKCluster) SetMonitoring(l bool) {
	c.modelCluster.Monitoring = l
}

// GetServiceMesh returns true if service mesh is enabled on the cluster
func (c *ACKCluster) GetServiceMesh() bool {
	return c.modelCluster.ServiceMesh
}

// SetServiceMesh sets service mesh flag on the cluster
func (c *ACKCluster) SetServiceMesh(m bool) {
	c.modelCluster.ServiceMesh = m
}

// getScaleOptionsFromModelV1 returns scale options for the cluster
func (c *ACKCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.modelCluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *ACKCluster) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.modelCluster.ScaleOptions, scaleOptions)
}

func (*ACKCluster) RequiresSshPublicKey() bool {
	return true
}

// GetTTL retrieves the TTL of the cluster
func (c *ACKCluster) GetTTL() time.Duration {
	return time.Duration(c.modelCluster.TtlMinutes) * time.Minute
}

// SetTTL sets the lifespan of a cluster
func (c *ACKCluster) SetTTL(ttl time.Duration) {
	c.modelCluster.TtlMinutes = uint(ttl.Minutes())
}

func (c *ACKCluster) ListNodeNames() (pkgCommon.NodeNames, error) {
	essClient, err := c.GetAlibabaESSClient(nil)
	if err != nil {
		return nil, err
	}
	request := ess.CreateDescribeScalingInstancesRequest()
	request.SetScheme(requests.HTTPS)
	request.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, c.modelCluster.ACK.RegionID))
	request.SetContentType(requests.Json)

	nodes := make(pkgCommon.NodeNames, 0)
	for _, nodepool := range c.modelCluster.ACK.NodePools {
		request.ScalingGroupId = nodepool.AsgID
		request.ScalingConfigurationId = nodepool.ScalingConfigID
		response, err := essClient.DescribeScalingInstances(request)
		if err != nil {
			return nil, emperror.WrapWith(err, "error listing nodepool instances", "scalingGroupName", nodepool.AsgID)
		}
		var instances []string
		for _, instance := range response.ScalingInstances.ScalingInstance {
			instances = append(instances, fmt.Sprint(c.modelCluster.ACK.RegionID, ".", instance.InstanceId))
		}
		nodes[nodepool.Name] = instances
	}
	return nodes, nil
}

// NodePoolExists returns true if node pool with nodePoolName exists
func (c *ACKCluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range c.modelCluster.ACK.NodePools {
		if np != nil && np.Name == nodePoolName {
			return true
		}
	}
	return false
}

// GetAlibabaCSClient creates an Alibaba Container Service client with the credentials
func (c *ACKCluster) GetAlibabaCSClient(cfg *sdk.Config) (*cs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}
	client, err := createAlibabaCSClient(cred, c.modelCluster.ACK.RegionID, cfg)
	return client, emperror.With(err, "cluster", c.modelCluster.Name)
}

// GetAlibabaECSClient creates an Alibaba Elastic Compute Service client with the credentials
func (c *ACKCluster) GetAlibabaECSClient(cfg *sdk.Config) (*ecs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	client, err := createAlibabaECSClient(cred, c.modelCluster.ACK.RegionID, cfg)
	return client, emperror.With(err, "cluster", c.modelCluster.Name)
}

// GetAlibabaESSClient creates an Alibaba Auto Scaling Service client with credentials
func (c *ACKCluster) GetAlibabaESSClient(cfg *sdk.Config) (*ess.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	client, err := createAlibabaESSClient(cred, c.modelCluster.ACK.RegionID, cfg)
	return client, emperror.With(err, "cluster", c.modelCluster.Name)
}

// GetAlibabaVPCClient creates an Alibaba Virtual Private Cloud client with credentials
func (c *ACKCluster) GetAlibabaVPCClient(cfg *sdk.Config) (*vpc.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	client, err := createAlibabaVPCClient(cred, c.modelCluster.ACK.RegionID, cfg)
	return client, emperror.With(err, "cluster", c.modelCluster.Name)
}

func createACKNodePoolsFromRequest(pools ack.NodePools, userId uint) ([]*model.ACKNodePoolModel, error) {
	nodePoolsCount := len(pools)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	var res = make([]*model.ACKNodePoolModel, len(pools))
	var i int
	for name, pool := range pools {
		res[i] = &model.ACKNodePoolModel{
			CreatedBy:    userId,
			Name:         name,
			InstanceType: pool.InstanceType,
			MinCount:     pool.MinCount,
			MaxCount:     pool.MaxCount,
			Count:        pool.MinCount,
			Labels:       pool.Labels,
		}
		i++
	}

	return res, nil
}

func (c *ACKCluster) createACKNodePoolsModelFromUpdateRequestData(pools ack.NodePools, userId uint) ([]*model.ACKNodePoolModel, error) {
	currentNodePoolMap := make(map[string]*model.ACKNodePoolModel, len(c.modelCluster.ACK.NodePools))
	updatedNodePools := make([]*model.ACKNodePoolModel, 0, len(pools))

	for _, nodePool := range c.modelCluster.ACK.NodePools {
		//Collect stored node pool info from DB
		currentNodePoolMap[nodePool.Name] = nodePool

		// Delete node pool stored in the DB but deleted with Update
		if pools[nodePool.Name] == nil {
			updatedNodePools = append(updatedNodePools, &model.ACKNodePoolModel{
				ID:              nodePool.ID,
				CreatedBy:       nodePool.CreatedBy,
				CreatedAt:       nodePool.CreatedAt,
				ClusterID:       nodePool.ClusterID,
				Name:            nodePool.Name,
				AsgID:           nodePool.AsgID,
				ScalingConfigID: nodePool.ScalingConfigID,
				Delete:          true,
			})
		}
	}

	for nodePoolName, nodePool := range pools {
		if currentNodePoolMap[nodePoolName] != nil {
			if currentNodePoolMap[nodePoolName].MinCount != nodePool.MinCount ||
				currentNodePoolMap[nodePoolName].MaxCount != nodePool.MaxCount ||
				currentNodePoolMap[nodePoolName].InstanceType != nodePool.InstanceType {
				updatedNodePools = append(updatedNodePools, &model.ACKNodePoolModel{
					ID:              currentNodePoolMap[nodePoolName].ID,
					CreatedBy:       currentNodePoolMap[nodePoolName].CreatedBy,
					CreatedAt:       currentNodePoolMap[nodePoolName].CreatedAt,
					ClusterID:       currentNodePoolMap[nodePoolName].ClusterID,
					Name:            nodePoolName,
					InstanceType:    nodePool.InstanceType,
					MinCount:        nodePool.MinCount,
					MaxCount:        nodePool.MaxCount,
					Count:           currentNodePoolMap[nodePoolName].Count,
					AsgID:           currentNodePoolMap[nodePoolName].AsgID,
					ScalingConfigID: currentNodePoolMap[nodePoolName].ScalingConfigID,
					Delete:          false,
				})
			}
		} else {
			// add new node pool

			// ---- [ Node instanceType check ] ---- //
			if len(nodePool.InstanceType) == 0 {
				c.log.Errorf("instanceType is missing for nodePool %v", nodePoolName)
				return nil, pkgErrors.ErrorInstancetypeFieldIsEmpty
			}

			updatedNodePools = append(updatedNodePools, &model.ACKNodePoolModel{
				CreatedBy:    userId,
				Name:         nodePoolName,
				InstanceType: nodePool.InstanceType,
				MinCount:     nodePool.MinCount,
				MaxCount:     nodePool.MaxCount,
				Count:        nodePool.MinCount,
				Delete:       false,
			})
		}

	}
	return updatedNodePools, nil
}

//CreateACKClusterFromModel creates ClusterModel struct from the Alibaba model
func CreateACKClusterFromModel(clusterModel *model.ClusterModel) (*ACKCluster, error) {
	alibabaCluster := ACKCluster{
		modelCluster: clusterModel,
		log:          log.WithField("cluster", clusterModel.Name),
	}
	return &alibabaCluster, nil
}

func CreateACKClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*ACKCluster, error) {
	cluster := ACKCluster{
		log: log.WithField("cluster", request.Name),
	}

	nodePools, err := createACKNodePoolsFromRequest(request.Properties.CreateClusterACK.NodePools, userId)
	if err != nil {
		return nil, err
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Properties.CreateClusterACK.ZoneID,
		Cloud:          request.Cloud,
		Distribution:   pkgCluster.ACK,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		ACK: model.ACKClusterModel{
			RegionID:                 request.Properties.CreateClusterACK.RegionID,
			ZoneID:                   request.Properties.CreateClusterACK.ZoneID,
			MasterInstanceType:       request.Properties.CreateClusterACK.MasterInstanceType,
			MasterSystemDiskCategory: request.Properties.CreateClusterACK.MasterSystemDiskCategory,
			MasterSystemDiskSize:     request.Properties.CreateClusterACK.MasterSystemDiskSize,
			SNATEntry:                true,
			SSHFlags:                 true,
			NodePools:                nodePools,
			VSwitchID:                request.Properties.CreateClusterACK.VSwitchID,
		},
		CreatedBy:  userId,
		TtlMinutes: request.TtlMinutes,
	}
	updateScaleOptions(&cluster.modelCluster.ScaleOptions, request.ScaleOptions)

	return &cluster, nil
}
func (c *ACKCluster) CreateCluster() error {
	c.log.Info("Start create cluster (Alibaba)")

	csClient, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	ecsClient, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	essClient, err := c.GetAlibabaESSClient(nil)
	if err != nil {
		return err
	}

	c.modelCluster.RbacEnabled = true
	vpcID, err := c.getVPCID()
	if err != nil {
		return emperror.Wrap(err, "failed to retreive VPC ID")
	}

	context := action.NewACKContext("", csClient, ecsClient, essClient)

	// All worker related fields are same as master ones to avoid instance is not available in that region
	// worker related fields are unused ones because we are asking 0 worker node with that request
	clusterContext := action.NewACKClusterCreationContext(
		*context,
		ack.AlibabaClusterCreateParams{
			ClusterType:              "Kubernetes",
			Name:                     c.modelCluster.Name,
			RegionID:                 c.modelCluster.ACK.RegionID,                 // "eu-central-1"
			ZoneID:                   c.modelCluster.ACK.ZoneID,                   // "eu-central-1a"
			MasterInstanceType:       c.modelCluster.ACK.MasterInstanceType,       // "ecs.sn1.large",
			MasterSystemDiskCategory: c.modelCluster.ACK.MasterSystemDiskCategory, // "cloud_efficiency",
			MasterSystemDiskSize:     c.modelCluster.ACK.MasterSystemDiskSize,     // 40,
			WorkerInstanceType:       c.modelCluster.ACK.MasterInstanceType,       // "ecs.sn1.large",
			WorkerSystemDiskCategory: "cloud_efficiency",                          // "cloud_efficiency",
			KeyPair:                  c.modelCluster.Name,                         // uploaded keyPair name
			NumOfNodes:               0,                                           // 0 (to make sure node pools are created properly),
			SNATEntry:                c.modelCluster.ACK.SNATEntry,                // true,
			SSHFlags:                 c.modelCluster.ACK.SSHFlags,                 // true,
			DisableRollback:          true,
			VPCID:                    vpcID,
			VSwitchID:                c.modelCluster.ACK.VSwitchID,
			ContainerCIDR:            "172.19.0.0/20",
			ServiceCIDR:              "172.16.0.0/16",
		},
	)

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return err
	}

	actions := []utils.Action{
		action.NewUploadSSHKeyAction(c.log, clusterContext, clusterSshSecret),
		action.NewCreateACKClusterAction(c.log, clusterContext),
		action.NewCreateACKNodePoolAction(c.log, c.modelCluster.ACK.NodePools, context, c.modelCluster.ACK.RegionID),
	}

	resp, err := utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, true)
	c.modelCluster.ACK.ProviderClusterID = clusterContext.ClusterID
	if err != nil {
		return emperror.WrapWith(err, "failed to create ACK cluster", "cluster", c.modelCluster.Name)
	}
	castedValue, ok := resp.(*ack.AlibabaDescribeClusterResponse)
	if !ok {
		return emperror.With(errors.New("could not cast cluster create response"), "cluster", c.modelCluster.Name)
	}
	c.modelCluster.ACK.KubernetesVersion = castedValue.KubernetesVersion
	c.alibabaCluster = castedValue

	kubeConfig, err := c.DownloadK8sConfig()
	if err != nil {
		return err
	}

	restKubeConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return emperror.With(err, "cluster", c.modelCluster.Name)
	}

	kubeClient, err := kubernetes.NewForConfig(restKubeConfig)
	if err != nil {
		return emperror.WrapWith(err, "could not generate kubeClient from config", "cluster", c.modelCluster.Name)
	}

	// create default storage class
	// TODO change this storagev1.VolumeBindingImmediate to storagev1.VolumeBindingWaitForFirstConsumer
	// when Alibaba supports this feature
	err = createDefaultStorageClass(kubeClient, "alicloud/disk", storagev1.VolumeBindingImmediate)
	if err != nil {
		return emperror.With(err, "cluster", c.modelCluster.Name)
	}

	return c.modelCluster.Save()
}

func (c *ACKCluster) getVPCID() (string, error) {
	if c.modelCluster.ACK.VSwitchID == "" {
		return "", nil
	}

	vpcClient, err := c.GetAlibabaVPCClient(nil)
	if err != nil {
		return "", emperror.Wrap(err, "failed to get Alibaba VPC client")
	}

	req := vpc.CreateDescribeVSwitchesRequest()
	req.VSwitchId = c.modelCluster.ACK.VSwitchID
	res, err := vpcClient.DescribeVSwitches(req)
	if err != nil {
		return "", emperror.WrapWith(err, "could not get VSwitch details", "vswitch", c.modelCluster.ACK.VSwitchID)
	}
	if len(res.VSwitches.VSwitch) != 1 {
		return "", errors.New("VSwitch not found")
	}
	return res.VSwitches.VSwitch[0].VpcId, nil
}

type alibabaConnectionInfo struct {
	JumpHost    string
	IntranetURI string
	InternetURI string
}

func getConnectionInfo(client *cs.Client, clusterID string) (inf alibabaConnectionInfo, err error) {
	details, err := action.GetClusterDetails(client, clusterID)
	if err != nil {
		return
	}
	for _, v := range details.Outputs {
		if v.OutputKey == "JumpHost" {
			if jh, ok := v.OutputValue.(string); ok {
				inf.JumpHost = jh
			}
		}
		if v.OutputKey == "APIServerIntranet" {
			if intra, ok := v.OutputValue.(string); ok {
				inf.IntranetURI = intra
			}
		}
		if v.OutputKey == "APIServerInternet" {
			if inter, ok := v.OutputValue.(string); ok {
				inf.InternetURI = inter
			}
		}
	}
	if inf.JumpHost == "" {
		err = errors.New("JumpHost not found")
		return
	}
	if inf.IntranetURI == "" {
		err = errors.New("IntranetURI not found")
		return
	}
	if inf.InternetURI == "" {
		err = errors.New("InternetURI not found")
		return
	}

	return
}

// Persist
// Deprecated: Do not use.
func (c *ACKCluster) Persist() error {
	return emperror.Wrap(c.modelCluster.Save(), "failed to persist cluster")
}

func (c *ACKCluster) DownloadK8sConfig() ([]byte, error) {
	cfg := sdk.NewConfig()
	cfg.AutoRetry = false
	cfg.Debug = true
	cfg.Timeout = time.Minute

	csClient, err := c.GetAlibabaCSClient(cfg)
	if err != nil {
		return nil, err
	}

	info, err := getConnectionInfo(csClient, c.modelCluster.ACK.ProviderClusterID)
	if err != nil {
		return nil, emperror.With(err, "cluster", c.modelCluster.Name)
	}
	sshHost := info.JumpHost

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return nil, err
	}
	sshKey := secret.NewSSHKeyPair(clusterSshSecret)

	signer, err := ssh.ParsePrivateKey([]byte(sshKey.PrivateKeyData))
	if err != nil {
		return nil, emperror.With(err, "cluster", c.modelCluster.Name)
	}
	clientConfig := ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshClient, err := ssh.Dial("tcp", fmt.Sprint(sshHost, ":22"), &clientConfig)
	if err != nil {
		return nil, emperror.With(err, "cluster", c.modelCluster.Name)
	}
	defer sshClient.Close()
	var buff bytes.Buffer
	w := bufio.NewWriter(&buff)
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, emperror.With(err, "cluster", c.modelCluster.Name)
	}
	defer sshSession.Close()
	sshSession.Stdout = w
	sshSession.Run(fmt.Sprintf("cat %s", "/etc/kubernetes/kube.conf"))
	w.Flush()
	return buff.Bytes(), emperror.With(err, "cluster", c.modelCluster.Name)

}

// GetCloud returns the cloud type of the cluster
func (c *ACKCluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *ACKCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

func (c *ACKCluster) GetName() string {
	return c.modelCluster.Name
}

func (c *ACKCluster) GetType() string {
	return c.modelCluster.Cloud
}

func (c *ACKCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.ACK.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				InstanceType:      np.InstanceType,
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				MinCount:          np.MinCount,
				MaxCount:          np.MaxCount,
				Count:             np.Count,
				Labels:            np.Labels,
			}
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.modelCluster.Status,
		StatusMessage:     c.modelCluster.StatusMessage,
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		Cloud:             c.modelCluster.Cloud,
		Distribution:      c.modelCluster.Distribution,
		ResourceID:        c.modelCluster.ID,
		Logging:           c.GetLogging(),
		Monitoring:        c.GetMonitoring(),
		ServiceMesh:       c.GetServiceMesh(),
		SecurityScan:      c.GetSecurityScan(),
		NodePools:         nodePools,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Region:            c.modelCluster.ACK.RegionID,
		TtlMinutes:        c.modelCluster.TtlMinutes,
		StartedAt:         c.modelCluster.StartedAt,
	}, nil
}

func (c *ACKCluster) DeleteCluster() error {
	c.log.Info("Start deleting cluster (Alibaba)")

	csClient, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	ecsClient, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	essClient, err := c.GetAlibabaESSClient(nil)
	if err != nil {
		return err
	}

	deleteContext := action.NewACKClusterDeletionContext(
		csClient,
		ecsClient,
		essClient,
		c.modelCluster.ACK.ProviderClusterID,
		c.modelCluster.ACK.NodePools,
		c.modelCluster.Name,
		c.modelCluster.ACK.RegionID)

	actions := []utils.Action{
		action.NewDeleteACKNodePoolAction(c.log, deleteContext),
		action.NewDeleteACKClusterAction(c.log, deleteContext),
		action.NewDeleteSSHKeyAction(c.log, deleteContext, c.modelCluster.Name, c.modelCluster.ACK.RegionID),
	}

	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, false)
	if err != nil {
		return emperror.WrapWith(err, "could not delete Alibaba cluster", "cluster", c.modelCluster.Name)
	}

	return nil
}

func (c *ACKCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userId uint) error {
	c.log.Info("Start updating cluster (Alibaba)")

	csClient, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	ecsClient, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	essClient, err := c.GetAlibabaESSClient(nil)
	if err != nil {
		return err
	}

	nodePoolModels, err := c.createACKNodePoolsModelFromUpdateRequestData(request.ACK.NodePools, userId)
	if err != nil {
		return err
	}

	var nodePoolsToCreate []*model.ACKNodePoolModel
	var nodePoolsToUpdate []*model.ACKNodePoolModel
	var nodePoolsToDelete []*model.ACKNodePoolModel

	for _, nodePool := range nodePoolModels {
		// delete nodePool
		if nodePool.Delete {
			c.log.Infof("nodePool %v will be deleted", nodePool.Name)
			nodePoolsToDelete = append(nodePoolsToDelete, nodePool)
			continue
		}
		// create nodePool
		if nodePool.ScalingConfigID == "" && nodePool.AsgID == "" {
			c.log.Infof("nodePool %v will be created", nodePool.Name)
			nodePoolsToCreate = append(nodePoolsToCreate, nodePool)
			continue
		} else {
			// update nodePool
			c.log.Infof("nodePool %v will be updated", nodePool.Name)
			nodePoolsToUpdate = append(nodePoolsToUpdate, nodePool)
		}
	}

	context := action.NewACKContext(c.modelCluster.ACK.ProviderClusterID, csClient, ecsClient, essClient)
	deleteContext := action.NewACKClusterDeletionContext(
		csClient,
		ecsClient,
		essClient,
		c.modelCluster.ACK.ProviderClusterID,
		nodePoolsToDelete,
		c.modelCluster.Name,
		c.modelCluster.ACK.RegionID)

	actions := []utils.Action{
		action.NewDeleteACKNodePoolAction(c.log, deleteContext),
		action.NewUpdateACKNodePoolAction(c.log, c.modelCluster.Name, nodePoolsToUpdate, context, c.modelCluster.ACK.RegionID),
		action.NewCreateACKNodePoolAction(c.log, nodePoolsToCreate, context, c.modelCluster.ACK.RegionID),
	}

	resp, err := utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, false)
	if err != nil {
		return emperror.WrapWith(err, "failed to update ACK cluster", "cluster", c.modelCluster.Name)
	}

	castedValue, ok := resp.(*ack.AlibabaDescribeClusterResponse)
	if !ok {
		return emperror.With(errors.New("could not cast cluster update response"), "cluster", c.modelCluster.Name)
	}

	c.modelCluster.ACK.NodePools = nodePoolModels
	c.alibabaCluster = castedValue

	return nil
}

// UpdateNodePools updates nodes pools of a cluster
func (c *ACKCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	return nil
}

func (c *ACKCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *ACKCluster) GetUID() string {
	return c.modelCluster.UID
}

func (c *ACKCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

func (c *ACKCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// GetLocation gets where the cluster is.
func (c *ACKCluster) GetLocation() string {
	return c.modelCluster.Location
}

func (c *ACKCluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

func (c *ACKCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

func (c *ACKCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality

	preNodePools := make(map[string]*ack.NodePool)
	for _, preNp := range c.modelCluster.ACK.NodePools {
		preNodePools[preNp.Name] = &ack.NodePool{
			InstanceType: preNp.InstanceType,
			MinCount:     preNp.MinCount,
			MaxCount:     preNp.MaxCount,
		}
	}

	preCl := &ack.UpdateClusterACK{
		NodePools: preNodePools,
	}

	c.log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.ACK, preCl)
}

func (c *ACKCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {
}

func (c *ACKCluster) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}

	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return "", err
	}
	inf, err := getConnectionInfo(client, c.modelCluster.ACK.ProviderClusterID)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(inf.InternetURI)
	if err != nil {
		return "", err
	}
	c.APIEndpoint = u.Host
	return c.APIEndpoint, nil
}

func (c *ACKCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}

func (c *ACKCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// SetStatus sets the cluster's status
func (c *ACKCluster) SetStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *ACKCluster) IsReady() (bool, error) {
	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return false, err
	}

	r, err := action.GetClusterDetails(client, c.modelCluster.ACK.ProviderClusterID)
	if err != nil {
		return false, err
	}

	return r.State == ack.AlibabaClusterStateRunning, nil
}

func interfaceArrayToStringArray(in []interface{}) (out []string) {
	out = make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (c *ACKCluster) validateRegion(regionID string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeRegionsRequest()
	req.SetScheme(requests.HTTPS)
	resp, err := client.DescribeRegions(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Regions.Region[].RegionId", content)
	if err != nil {
		return err
	}

	var validRegions []string
	if r, ok := items.([]interface{}); ok {
		validRegions = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(regionID, validRegions) {
		return errors.New("Invalid region (" + regionID + ") specified, must be one of: " + strings.Join(validRegions, ", "))
	}

	return nil
}

func (c *ACKCluster) validateZone(regionID, zoneID string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeZonesRequest()
	req.SetScheme(requests.HTTPS)
	req.RegionId = regionID
	resp, err := client.DescribeZones(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Zones.Zone[].ZoneId", content)
	if err != nil {
		return err
	}

	var validZones []string
	if r, ok := items.([]interface{}); ok {
		validZones = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(zoneID, validZones) {
		return errors.New("Invalid region (" + zoneID + ") specified, must be one of: " + strings.Join(validZones, ", "))
	}

	return nil
}

func (c *ACKCluster) validateInstanceType(regionID, zoneID, instanceType string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeZonesRequest()
	req.SetScheme(requests.HTTPS)
	req.RegionId = regionID
	resp, err := client.DescribeZones(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Zones.Zone[?ZoneId == '"+zoneID+"'].AvailableResources.ResourcesInfo[].InstanceTypes.supportedInstanceType[]", content)
	if err != nil {
		return err
	}

	var validInstanceTypes []string
	if r, ok := items.([]interface{}); ok {
		validInstanceTypes = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(instanceType, validInstanceTypes) {
		return errors.New("Invalid instance_type (" + instanceType + ") specified, must be one of: " + strings.Join(validInstanceTypes, ", "))
	}

	return nil
}

func (c *ACKCluster) validateSystemDiskCategories(regionID, zoneID, diskCategory string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeZonesRequest()
	req.SetScheme(requests.HTTPS)
	req.RegionId = regionID
	resp, err := client.DescribeZones(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Zones.Zone[?ZoneId == '"+zoneID+"'].AvailableResources.ResourcesInfo[].SystemDiskCategories.supportedSystemDiskCategory[]", content)
	if err != nil {
		return err
	}

	var validDiskCategory []string
	if r, ok := items.([]interface{}); ok {
		validDiskCategory = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(diskCategory, validDiskCategory) {
		return errors.New("Invalid disk_category (" + diskCategory + ") specified, must be one of: " + strings.Join(validDiskCategory, ", "))
	}

	return nil
}

func (c *ACKCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	var (
		region       = r.Properties.CreateClusterACK.RegionID
		zone         = r.Properties.CreateClusterACK.ZoneID
		instanceType = r.Properties.CreateClusterACK.MasterInstanceType
		diskCategory = r.Properties.CreateClusterACK.MasterSystemDiskCategory
	)
	err := c.validateRegion(region)
	if err != nil {
		return err
	}

	err = c.validateZone(region, zone)
	if err != nil {
		return err
	}

	err = c.validateInstanceType(region, zone, instanceType)
	if err != nil {
		return err
	}

	err = c.validateSystemDiskCategories(region, zone, diskCategory)
	if err != nil {
		return err
	}

	for _, np := range r.Properties.CreateClusterACK.NodePools {
		var (
			instanceType = np.InstanceType
			//diskCategory = np.SystemDiskCategory
		)

		err = c.validateInstanceType(region, zone, instanceType)
		if err != nil {
			return err
		}

		err = c.validateSystemDiskCategories(region, zone, diskCategory)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACKCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

func (c *ACKCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

func (c *ACKCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetK8sIpv4Cidrs returns possible IP ranges for pods and services in the cluster
// On ACK the services and pods IP ranges can be fetched from Alibaba
func (c *ACKCluster) GetK8sIpv4Cidrs() (*pkgCluster.Ipv4Cidrs, error) {
	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get alibaba CS client")
	}

	cluster, err := action.GetClusterDetails(client, c.modelCluster.ACK.ProviderClusterID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster details")
	}

	return &pkgCluster.Ipv4Cidrs{
		ServiceClusterIPRanges: []string{cluster.Parameters.ServiceCIDR},
		PodIPRanges:            []string{cluster.SubnetCIDR},
	}, nil
}

func (c *ACKCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func (c *ACKCluster) createAlibabaCredentialsFromSecret() (*credentials.AccessKeyCredential, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to create alibaba creds from secret", "cluster", c.modelCluster.Name)
	}
	return verify.CreateAlibabaCredentials(clusterSecret.Values), nil
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *ACKCluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *ACKCluster) GetKubernetesUserName() (string, error) {
	return "", nil
}

func createAlibabaConfig() *sdk.Config {
	return sdk.NewConfig().
		WithAutoRetry(true).
		WithDebug(true).WithTimeout(time.Minute)
}

func createAlibabaCSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*cs.Client, error) {
	if cfg == nil {
		cfg = createAlibabaConfig()
	}

	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	client, err := cs.NewClientWithOptions(regionID, cfg, cred)
	client.SetReadTimeout(ack.ACKRequestReadTimeout)
	return client, emperror.Wrap(err, "could not create Alibaba CSClient")
}

func createAlibabaECSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*ecs.Client, error) {
	if cfg == nil {
		cfg = createAlibabaConfig()
	}

	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	client, err := ecs.NewClientWithOptions(regionID, cfg, cred)
	client.SetReadTimeout(ack.ACKRequestReadTimeout)
	return client, emperror.Wrap(err, "could not create Alibaba ECSClient")
}

func createAlibabaESSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*ess.Client, error) {
	if cfg == nil {
		cfg = createAlibabaConfig()
	}
	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	client, err := ess.NewClientWithOptions(regionID, cfg, cred)
	client.SetReadTimeout(ack.ACKRequestReadTimeout)
	return client, emperror.Wrap(err, "could not create Alibaba ESSClient")
}

func createAlibabaVPCClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*vpc.Client, error) {
	if cfg == nil {
		cfg = createAlibabaConfig()
	}
	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	client, err := vpc.NewClientWithOptions(regionID, cfg, cred)
	client.SetReadTimeout(ack.ACKRequestReadTimeout)
	return client, emperror.Wrap(err, "could not create Alibaba VPCClient")
}

// GetCreatedBy returns cluster create userID.
func (c *ACKCluster) GetCreatedBy() uint {
	return c.modelCluster.CreatedBy
}
