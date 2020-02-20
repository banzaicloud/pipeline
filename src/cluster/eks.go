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
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/banzaicloud/pipeline/internal/global"

	"github.com/banzaicloud/pipeline/pkg/k8sutil"

	"github.com/banzaicloud/pipeline/pkg/cluster/eks/nodepools"

	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/sirupsen/logrus"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

const asgWaitLoopSleepSeconds = 5
const asgFulfillmentTimeout = 10 * time.Minute

// CreateEKSClusterFromRequest creates ClusterModel struct from the request
func CreateEKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EKSCluster, error) {
	cluster := EKSCluster{
		log: log.WithField("cluster", request.Name),
	}

	modelNodePools := createNodePoolsFromRequest(request.Properties.CreateClusterEKS.NodePools, userId)

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.EKS,
		EKS: model.EKSClusterModel{
			Version:               request.Properties.CreateClusterEKS.Version,
			LogTypes:              request.Properties.CreateClusterEKS.LogTypes,
			NodePools:             modelNodePools,
			VpcId:                 &request.Properties.CreateClusterEKS.Vpc.VpcId,
			VpcCidr:               &request.Properties.CreateClusterEKS.Vpc.Cidr,
			RouteTableId:          &request.Properties.CreateClusterEKS.RouteTableId,
			Subnets:               createSubnetsFromRequest(request.Properties.CreateClusterEKS),
			DefaultUser:           request.Properties.CreateClusterEKS.IAM.DefaultUser,
			ClusterRoleId:         request.Properties.CreateClusterEKS.IAM.ClusterRoleID,
			NodeInstanceRoleId:    request.Properties.CreateClusterEKS.IAM.NodeInstanceRoleID,
			APIServerAccessPoints: createAPIServerAccesPointsFromRequest(request),
		},
		RbacEnabled: true,
		CreatedBy:   userId,
	}

	updateScaleOptions(&cluster.modelCluster.ScaleOptions, request.ScaleOptions)

	// subnet mapping
	cluster.SubnetMapping = createSubnetMappingFromRequest(request.Properties.CreateClusterEKS)

	return &cluster, nil
}

func createAPIServerAccesPointsFromRequest(request *pkgCluster.CreateClusterRequest) []string {
	if len(request.Properties.CreateClusterEKS.APIServerAccessPoints) != 0 {
		return request.Properties.CreateClusterEKS.APIServerAccessPoints
	}
	return []string{"public"}
}

func createNodePoolsFromRequest(nodePools map[string]*pkgEks.NodePool, userId uint) []*model.AmazonNodePoolsModel {
	var modelNodePools = make([]*model.AmazonNodePoolsModel, len(nodePools))
	i := 0
	for nodePoolName, nodePool := range nodePools {
		modelNodePools[i] = &model.AmazonNodePoolsModel{
			CreatedBy:        userId,
			Name:             nodePoolName,
			NodeSpotPrice:    nodePool.SpotPrice,
			Autoscaling:      nodePool.Autoscaling,
			NodeMinCount:     nodePool.MinCount,
			NodeMaxCount:     nodePool.MaxCount,
			Count:            nodePool.Count,
			NodeImage:        nodePool.Image,
			NodeInstanceType: nodePool.InstanceType,
			Labels:           nodePool.Labels,
			Delete:           false,
		}
		i++
	}
	return modelNodePools
}

// createSubnetsFromRequest collects distinct existing (subnetid !=0) and to be created subnets from the request
func createSubnetsFromRequest(eksRequest *pkgEks.CreateClusterEKS) []*model.EKSSubnetModel {
	if eksRequest == nil {
		return nil
	}

	var subnetsFromRequest []*pkgEks.Subnet
	for _, subnet := range eksRequest.Subnets {
		if subnet != nil {
			subnetsFromRequest = append(subnetsFromRequest, subnet)
		}
	}

	for _, np := range eksRequest.NodePools {
		if np != nil && np.Subnet != nil {
			subnetsFromRequest = append(subnetsFromRequest, np.Subnet)
		}
	}

	uniqueSubnets := make(map[string]*model.EKSSubnetModel, 0)
	for _, subnet := range subnetsFromRequest {
		if subnet != nil {
			if subnet.SubnetId != "" {
				if _, ok := uniqueSubnets[subnet.SubnetId]; !ok {
					uniqueSubnets[subnet.SubnetId] = &model.EKSSubnetModel{SubnetId: &subnet.SubnetId}
				}
			} else if subnet.Cidr != "" {
				if _, ok := uniqueSubnets[subnet.Cidr]; !ok {
					uniqueSubnets[subnet.Cidr] = &model.EKSSubnetModel{
						Cidr:             &subnet.Cidr,
						AvailabilityZone: &subnet.AvailabilityZone,
					}
				}
			}
		}
	}

	var modelSubnets []*model.EKSSubnetModel
	for _, subnet := range uniqueSubnets {
		modelSubnets = append(modelSubnets, subnet)
	}

	return modelSubnets
}

// createSubnetMappingFromRequest maps node pools to subnets provided in request.Properties.CreateClusterEKS.
// The subnets identified by the "default" key in the returned map represent the subnets provided in
// request.Properties.CreateClusterEKS.Subnets
func createSubnetMappingFromRequest(eksRequest *pkgEks.CreateClusterEKS) map[string][]*pkgEks.Subnet {
	subnetMapping := make(map[string][]*pkgEks.Subnet, len(eksRequest.NodePools)+1)

	subnetMapping["default"] = eksRequest.Subnets

	for nodePoolName, nodePool := range eksRequest.NodePools {
		subnetMapping[nodePoolName] = []*pkgEks.Subnet{nodePool.Subnet}
	}

	return subnetMapping
}

// EKSCluster struct for EKS cluster
type EKSCluster struct {
	modelCluster *model.ClusterModel

	// maps node pools to subnets. The subnets identified by the "default" key represent the subnets provided in
	// request.Properties.CreateClusterEKS.Subnets
	SubnetMapping map[string][]*pkgEks.Subnet
	log           logrus.FieldLogger
	CommonClusterBase
}

func (c *EKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	panic("not used")
}

func (c *EKSCluster) CreateCluster() error {
	panic("not used")
}

func (c *EKSCluster) DeleteCluster() error {
	panic("not used")
}

// Deprecated: UpdateCluster updates EKS cluster in cloud
func (c *EKSCluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest, updatedBy uint) error {
	panic("not used")
}

func (c *EKSCluster) GetEKSModel() *model.EKSClusterModel {
	return &c.modelCluster.EKS
}

func (c *EKSCluster) GetSubnetMapping() map[string][]*pkgEks.Subnet {
	return c.SubnetMapping
}

// GetOrganizationId gets org where the cluster belongs
func (c *EKSCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (c *EKSCluster) GetLocation() string {
	return c.modelCluster.Location
}

// GetSecretId retrieves the secret id
func (c *EKSCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

// GetSshSecretId retrieves the secret id
func (c *EKSCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (c *EKSCluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *EKSCluster) GetAPIEndpoint() (string, error) {
	config, err := c.GetK8sConfig()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster's Kubeconfig")
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
}

// CreateEKSClusterFromModel creates ClusterModel struct from the model
func CreateEKSClusterFromModel(clusterModel *model.ClusterModel) *EKSCluster {
	return &EKSCluster{
		modelCluster: clusterModel,
		log:          log.WithField("cluster", clusterModel.Name),
	}
}

func (c *EKSCluster) createAWSCredentialsFromSecret() (*credentials.Credentials, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return verify.CreateAWSCredentials(clusterSecret.Values), nil
}

func (c *EKSCluster) SetCurrentWorkflowID(workflowID string) error {
	return c.modelCluster.EKS.SetCurrentWorkflowID(workflowID)
}

func (c *EKSCluster) PersistSSHGenerate(sshGenerated bool) error {
	return c.modelCluster.EKS.PersistSSHGenerate(sshGenerated)
}

func (c *EKSCluster) IsSSHGenerated() bool {
	return c.modelCluster.EKS.IsSSHGenerated()
}

// Persist saves the cluster model
// Deprecated: Do not use.
func (c *EKSCluster) Persist() error {
	return errors.WrapIf(c.modelCluster.Save(), "failed to persist cluster")
}

// GetName returns the name of the cluster
func (c *EKSCluster) GetName() string {
	return c.modelCluster.Name
}

// GetCloud returns the cloud type of the cluster
func (c *EKSCluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *EKSCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

// UpdateNodePools updates nodes pools of a cluster
// 		This will become obsolete once we have the Node Pool API ready
func (c *EKSCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	c.log.Info("start updating node pools")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return errors.WrapIf(err, "error retrieving AWS credentials")
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return errors.WrapIf(err, "error creating AWS awsSession")
	}

	autoscalingSrv := autoscaling.New(awsSession)
	cloudformationSrv := cloudformation.New(awsSession)

	waitRoutines := 0
	waitChan := make(chan error)
	defer close(waitChan)

	var caughtErrors []error
	ASGWaitLoopCount := int(asgFulfillmentTimeout.Seconds() / asgWaitLoopSleepSeconds)

	for poolName, nodePool := range request.NodePools {

		asgName, err := c.getAutoScalingGroupName(cloudformationSrv, autoscalingSrv, poolName)
		if err != nil {
			c.log.Errorf("ASG not found for node pool %v. %v", poolName, err.Error())
			continue
		}
		params := &autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(*asgName),
			DesiredCapacity:      aws.Int64(int64(nodePool.Count)),
			HonorCooldown:        aws.Bool(false),
		}
		c.log.Infof("setting node pool %s size to %d", poolName, nodePool.Count)
		_, err = autoscalingSrv.SetDesiredCapacity(params)
		if err != nil {
			caughtErrors = append(caughtErrors, errors.Wrapf(err, "failed to set size for node pool %s", poolName))
			continue
		}
		c.setNodePoolSize(poolName, nodePool.Count)

		waitRoutines++
		go func(poolName string) {
			waitChan <- nodepools.WaitForASGToBeFulfilled(context.Background(), awsSession, c.log, c.modelCluster.Name,
				poolName, ASGWaitLoopCount, asgWaitLoopSleepSeconds*time.Second)
		}(poolName)

	}

	// wait for goroutines to finish
	for i := 0; i < waitRoutines; i++ {
		waitErr := <-waitChan
		if waitErr != nil {
			caughtErrors = append(caughtErrors, waitErr)
		}
	}

	return errors.Combine(caughtErrors...)
}

func (c *EKSCluster) getAutoScalingGroupName(cloudformationSrv *cloudformation.CloudFormation, autoscalingSrv *autoscaling.AutoScaling, nodePoolName string) (*string, error) {
	logResourceId := "NodeGroup"
	stackName := nodepools.GenerateNodePoolStackName(c.modelCluster.Name, nodePoolName)
	describeStackResourceInput := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: &logResourceId,
		StackName:         aws.String(stackName)}
	describeStacksOutput, err := cloudformationSrv.DescribeStackResource(describeStackResourceInput)
	if err != nil {
		return nil, err
	}

	return describeStacksOutput.StackResourceDetail.PhysicalResourceId, nil
}

// GetStatus describes the status of this EKS cluster.
func (c *EKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	var hasSpotNodePool bool

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.EKS.NodePools {
		if np != nil {

			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:       np.Autoscaling,
				Count:             np.Count,
				InstanceType:      np.NodeInstanceType,
				SpotPrice:         np.NodeSpotPrice,
				MinCount:          np.NodeMinCount,
				MaxCount:          np.NodeMaxCount,
				Image:             np.NodeImage,
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				Labels:            np.Labels,
			}
			if np.NodeSpotPrice != "" && np.NodeSpotPrice != "0" {
				hasSpotNodePool = true
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
		Spot:              hasSpotNodePool,
		ResourceID:        c.modelCluster.ID,
		NodePools:         nodePools,
		Version:           c.modelCluster.EKS.Version,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Region:            c.modelCluster.Location,
		StartedAt:         c.modelCluster.StartedAt,
	}, nil
}

// GetID returns the DB ID of this cluster
func (c *EKSCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *EKSCluster) GetUID() string {
	return c.modelCluster.UID
}

// GetModel returns the DB model of this cluster
func (c *EKSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

// CheckEqualityToUpdate validates the update request
func (c *EKSCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preNodePools := make(map[string]*pkgEks.NodePool)
	for _, preNp := range c.modelCluster.EKS.NodePools {

		preNodePools[preNp.Name] = &pkgEks.NodePool{
			InstanceType: preNp.NodeInstanceType,
			SpotPrice:    preNp.NodeSpotPrice,
			Autoscaling:  preNp.Autoscaling,
			MinCount:     preNp.NodeMinCount,
			MaxCount:     preNp.NodeMaxCount,
			Count:        preNp.Count,
			Image:        preNp.NodeImage,
		}
	}

	preCl := &pkgEks.UpdateClusterAmazonEKS{
		NodePools: preNodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.EKS, preCl)
}

// AddDefaultsToUpdate adds defaults to update request
func (c *EKSCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {
	defaultImage, _ := pkgEks.GetDefaultImageID(c.modelCluster.Location, c.modelCluster.EKS.Version)

	// add default node image(s) if needed
	if r != nil && r.EKS != nil && r.EKS.NodePools != nil {
		for _, np := range r.EKS.NodePools {
			if len(np.Image) == 0 {
				np.Image = defaultImage
			}
		}
	}
}

// DeleteFromDatabase deletes model from the database
func (c *EKSCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}

// SetStatus sets the cluster's status
func (c *EKSCluster) SetStatus(status string, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// NodePoolExists returns true if node pool with nodePoolName exists
func (c *EKSCluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range c.modelCluster.EKS.NodePools {
		if np != nil && np.Name == nodePoolName {
			return true
		}
	}
	return false
}

func (c *EKSCluster) setNodePoolSize(nodePoolName string, count int) bool {
	for _, np := range c.modelCluster.EKS.NodePools {
		if np != nil && np.Name == nodePoolName {
			np.Count = count
		}
	}
	return false
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *EKSCluster) IsReady() (bool, error) {
	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return false, err
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return false, err
	}

	eksSvc := eks.New(awsSession)
	describeCluster := &eks.DescribeClusterInput{Name: aws.String(c.GetName())}
	clusterDesc, err := eksSvc.DescribeCluster(describeCluster)
	if err != nil {
		return false, err
	}

	return aws.StringValue(clusterDesc.Cluster.Status) == eks.ClusterStatusActive, nil
}

// GetSecretWithValidation returns secret from vault
func (c *EKSCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// SaveConfigSecretId saves the config secret id in database
func (c *EKSCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId returns config secret id
func (c *EKSCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config for internal use
func (c *EKSCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// GetK8sUserConfig returns the Kubernetes config for external users
func (c *EKSCluster) GetK8sUserConfig() ([]byte, error) {
	adminConfig, err := c.CommonClusterBase.getConfig(c)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get raw kubernetes config")
	}

	if global.Config.Distribution.EKS.ExposeAdminKubeconfig {
		return adminConfig, nil
	}

	parsedAdminConfig, err := clientcmd.Load(adminConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to load kubernetes API config")
	}

	userConfig := k8sutil.ExtractConfigBase(parsedAdminConfig).CreateConfigFromTemplate(
		k8sutil.CreateAuthInfoFunc(func(clusterName string) *clientcmdapi.AuthInfo {
			return &clientcmdapi.AuthInfo{
				Exec: &clientcmdapi.ExecConfig{
					APIVersion: "client.authentication.k8s.io/v1alpha1",
					Command:    "aws-iam-authenticator",
					Args:       []string{"token", "-i", clusterName},
				},
			}
		}))

	out, err := clientcmd.Write(*userConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize generated user config")
	}

	return out, nil
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *EKSCluster) RequiresSshPublicKey() bool {
	return true
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *EKSCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetScaleOptions returns scale options for the cluster
func (c *EKSCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.modelCluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *EKSCluster) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.modelCluster.ScaleOptions, scaleOptions)
}

// GetEKSNodePools returns EKS node pools from a common cluster.
func GetEKSNodePools(cluster CommonCluster) ([]*model.AmazonNodePoolsModel, error) {
	ekscluster, ok := cluster.(*EKSCluster)
	if !ok {
		return nil, ErrInvalidClusterInstance
	}

	return ekscluster.modelCluster.EKS.NodePools, nil
}

func (c *EKSCluster) GetKubernetesVersion() (string, error) {
	return c.modelCluster.EKS.Version, nil
}
