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
	"errors"
	"fmt"
	"strings"
	"time"

	"emperror.dev/emperror"
	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cloudinfo"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks/action"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
)

const mapRolesTemplate = `- rolearn: %s
  username: system:node:{{EC2PrivateDNSName}}
  groups:
  - system:bootstrappers
  - system:nodes
`

const mapUsersTemplate = `- userarn: %s
  username: %s
  groups:
  - system:masters
`

const asgWaitLoopSleepSeconds = 5

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
			Version:      request.Properties.CreateClusterEKS.Version,
			NodePools:    modelNodePools,
			VpcId:        &request.Properties.CreateClusterEKS.Vpc.VpcId,
			VpcCidr:      &request.Properties.CreateClusterEKS.Vpc.Cidr,
			RouteTableId: &request.Properties.CreateClusterEKS.RouteTableId,
			Subnets:      createSubnetsFromRequest(request.Properties.CreateClusterEKS.Subnets),
		},
		CreatedBy:  userId,
		TtlMinutes: request.TtlMinutes,
	}

	updateScaleOptions(&cluster.modelCluster.ScaleOptions, request.ScaleOptions)
	return &cluster, nil
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

func createSubnetsFromRequest(subnets []*pkgEks.ClusterSubnet) []*model.EKSSubnetModel {
	var modelSubnets []*model.EKSSubnetModel
	for _, subnet := range subnets {
		if subnet != nil {
			modelSubnets = append(modelSubnets, &model.EKSSubnetModel{
				SubnetId: &subnet.SubnetId,
				Cidr:     &subnet.Cidr,
			})
		}
	}
	return modelSubnets
}

// EKSCluster struct for EKS cluster
type EKSCluster struct {
	modelCluster             *model.ClusterModel
	APIEndpoint              string
	CloudInfoClient          *cloudinfo.Client
	CertificateAuthorityData []byte
	awsAccessKeyID           string
	awsSecretAccessKey       string
	log                      logrus.FieldLogger
	CommonClusterBase
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
	return c.APIEndpoint, nil
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

// CreateCluster creates an EKS cluster with cloudformation templates.
func (c *EKSCluster) CreateCluster() error {
	c.log.Info("Start creating EKS cluster")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve AWS credentials from secret")
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return emperror.Wrap(err, "failed to create AWS session")
	}

	// role that controls access to resources for creating an EKS cluster
	eksStackName := c.generateStackNameForCluster()
	sshKeyName := c.generateSSHKeyNameForCluster()

	c.modelCluster.RbacEnabled = true

	log.Infoln("Getting CloudFormation template for creating node pools for EKS cluster")
	nodePoolTemplate, err := pkgEks.GetNodePoolTemplate()
	if err != nil {
		return emperror.Wrap(err, "failed to get CloudFormation template for node pools")
	}

	creationContext := action.NewEksClusterCreationContext(
		session,
		c.modelCluster.Name,
		sshKeyName,
		nodePoolTemplate,
	)

	sshSecret, err := c.getSshSecret(c)
	if err != nil {
		return emperror.Wrap(err, "failed to get ssh secret")
	}

	creationContext.VpcID = c.modelCluster.EKS.VpcId
	creationContext.RouteTableID = c.modelCluster.EKS.RouteTableId
	for _, subnet := range c.modelCluster.EKS.Subnets {
		if aws.StringValue(subnet.SubnetId) != "" {
			creationContext.SubnetIDs = append(creationContext.SubnetIDs, subnet.SubnetId)
		} else if aws.StringValue(subnet.Cidr) != "" {
			creationContext.SubnetBlocks = append(creationContext.SubnetBlocks, subnet.Cidr)
		}
	}

	creationContext.ScaleEnabled = c.GetScaleOptions() != nil && c.GetScaleOptions().Enabled
	ASGWaitLoopCount := int(viper.GetDuration(config.EksASGFulfillmentTimeout).Seconds() / asgWaitLoopSleepSeconds)
	headNodePoolName := viper.GetString(config.PipelineHeadNodePoolName)

	actions := []utils.Action{
		action.NewCreateVPCAndRolesAction(c.log, creationContext, eksStackName),
		action.NewCreateClusterUserAccessKeyAction(c.log, creationContext),
		action.NewPersistClusterUserAccessKeyAction(c.log, creationContext, c.GetOrganizationId()),
		action.NewUploadSSHKeyAction(c.log, creationContext, sshSecret),
		action.NewGenerateVPCConfigRequestAction(c.log, creationContext, eksStackName, c.GetOrganizationId()),
		action.NewCreateEksClusterAction(c.log, creationContext, c.modelCluster.EKS.Version),
		action.NewCreateUpdateNodePoolStackAction(c.log, true, creationContext, ASGWaitLoopCount, asgWaitLoopSleepSeconds*time.Second, headNodePoolName, c.modelCluster.EKS.NodePools...),
	}

	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, false)
	if err != nil {
		return emperror.Wrap(err, "failed to create EKS cluster")
	}

	c.APIEndpoint = aws.StringValue(creationContext.APIEndpoint)
	c.CertificateAuthorityData, err = base64.StdEncoding.DecodeString(aws.StringValue(creationContext.CertificateAuthorityData))

	if err != nil {
		return emperror.Wrap(err, "failed to base64 decode EKS K8S certificate authority data")
	}

	// Create the aws-auth ConfigMap for letting other nodes join, and users access the API
	// See: https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html
	bootstrapCredentials, _ := awsCred.Get()
	c.awsAccessKeyID = bootstrapCredentials.AccessKeyID
	c.awsSecretAccessKey = bootstrapCredentials.SecretAccessKey

	defer func() {
		c.awsAccessKeyID = creationContext.ClusterUserAccessKeyId
		c.awsSecretAccessKey = creationContext.ClusterUserSecretAccessKey
		// AWS needs some time to distribute the access key to every service
		time.Sleep(15 * time.Second)
	}()

	kubeConfig, err := c.DownloadK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve K8S config")
	}

	restKubeConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create K8S config object")
	}

	kubeClient, err := kubernetes.NewForConfig(restKubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create K8S client")
	}

	constraint, err := semver.NewConstraint(">= 1.12")
	if err != nil {
		return emperror.Wrap(err, "could not set  1.12 constraint for semver")
	}
	kubeVersion, err := semver.NewVersion(c.modelCluster.EKS.Version)
	if err != nil {
		return emperror.Wrap(err, "could not set eks version for semver check")
	}
	var volumeBindingMode storagev1.VolumeBindingMode
	if constraint.Check(kubeVersion) {
		volumeBindingMode = storagev1.VolumeBindingWaitForFirstConsumer
	} else {
		volumeBindingMode = storagev1.VolumeBindingImmediate
	}

	storageClassConstraint, err := semver.NewConstraint("< 1.11")
	if err != nil {
		return emperror.Wrap(err, "could not set  1.11 constraint for semver")
	}

	if storageClassConstraint.Check(kubeVersion) {
		// create default storage class
		err = createDefaultStorageClass(kubeClient, "kubernetes.io/aws-ebs", volumeBindingMode, nil)
		if err != nil {
			return emperror.WrapWith(err, "failed to create default storage class",
				"provisioner", "kubernetes.io/aws-ebs",
				"bindingMode", volumeBindingMode)
		}
	}

	awsAuthConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-auth"},
		Data: map[string]string{
			"mapRoles": fmt.Sprintf(mapRolesTemplate, creationContext.NodeInstanceRoleArn),
			"mapUsers": fmt.Sprintf(mapUsersTemplate, creationContext.ClusterUserArn, creationContext.ClusterName),
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Create(&awsAuthConfigMap)
	if err != nil {
		return emperror.WrapWith(err, "failed to create config map", "configmap", awsAuthConfigMap.Name)
	}

	err = c.modelCluster.Save()
	if err != nil {
		return emperror.Wrap(err, "failed to persist cluster to database")
	}

	c.log.Info("EKS cluster created.")

	return nil
}

func (c *EKSCluster) generateSSHKeyNameForCluster() string {
	return "pipeline-eks-ssh-" + c.modelCluster.Name
}

func (c *EKSCluster) generateNodePoolStackName(nodePool *model.AmazonNodePoolsModel) string {
	return action.GenerateNodePoolStackName(c.modelCluster.Name, nodePool.Name)
}

func (c *EKSCluster) generateStackNameForCluster() string {
	return "pipeline-eks-" + c.modelCluster.Name
}

func (c *EKSCluster) generateIAMRoleNameForCluster() string {
	return "pipeline-eks-" + c.modelCluster.Name
}

// Persist saves the cluster model
// Deprecated: Do not use.
func (c *EKSCluster) Persist() error {
	return emperror.Wrap(c.modelCluster.Save(), "failed to persist cluster")
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

// DeleteCluster deletes cluster from EKS
func (c *EKSCluster) DeleteCluster() error {
	c.log.Info("Start delete EKS cluster")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return err
	}

	deleteContext := action.NewEksClusterDeleteContext(
		session,
		c.modelCluster.Name,
	)
	var actions []utils.Action
	actions = append(actions, action.NewWaitResourceDeletionAction(c.log, deleteContext)) // wait for ELBs to be deleted

	nodePoolStackNames := c.getNodepoolStackNamesToDelete(session)
	deleteNodePoolsAction := action.NewDeleteStacksAction(c.log, deleteContext, nodePoolStackNames...)

	actions = append(actions,
		deleteNodePoolsAction,
		action.NewDeleteClusterAction(c.log, deleteContext),
		action.NewDeleteSSHKeyAction(c.log, deleteContext, c.generateSSHKeyNameForCluster()),
		action.NewDeleteClusterUserAccessKeyAction(c.log, deleteContext),
		action.NewDeleteClusterUserAccessKeySecretAction(c.log, deleteContext, c.GetOrganizationId()),
		action.NewDeleteStacksAction(c.log, deleteContext, c.generateStackNameForCluster()),
	)
	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, false)
	if err != nil {
		c.log.Errorln("EKS cluster delete error:", err.Error())
		return err
	}

	return nil
}

func (c *EKSCluster) getNodepoolStackNamesToDelete(sess *session.Session) []string {
	stackNames := make([]string, 0)
	uniqueMap := make(map[string]bool, 0)

	for _, nodePool := range c.modelCluster.EKS.NodePools {
		nodePoolStackName := c.generateNodePoolStackName(nodePool)
		stackNames = append(stackNames, nodePoolStackName)
		uniqueMap[nodePoolStackName] = true
	}

	c.log.Debugf("stack names from DB: %+v", stackNames)

	tags := map[string]string{
		"pipeline-cluster-name": c.GetName(),
		"pipeline-stack-type":   "nodepool",
	}
	cfStackNames, err := pkgCloudformation.GetExistingTaggedStackNames(cloudformation.New(sess), tags)
	if err != nil {
		c.log.Error(err)
	} else {
		for _, stackName := range cfStackNames {
			if !uniqueMap[stackName] {
				stackNames = append(stackNames, stackName)
			}
		}
	}

	c.log.Debugf("stack names from DB + CF: %+v", stackNames)

	return stackNames
}

func (c *EKSCluster) createNodePoolsFromUpdateRequest(requestedNodePools map[string]*pkgEks.NodePool, userId uint) ([]*model.AmazonNodePoolsModel, error) {

	currentNodePoolMap := make(map[string]*model.AmazonNodePoolsModel, len(c.modelCluster.EKS.NodePools))
	for _, nodePool := range c.modelCluster.EKS.NodePools {
		currentNodePoolMap[nodePool.Name] = nodePool
	}

	updatedNodePools := make([]*model.AmazonNodePoolsModel, 0, len(requestedNodePools))

	for nodePoolName, nodePool := range requestedNodePools {
		if currentNodePoolMap[nodePoolName] != nil {
			// update existing node pool
			updatedNodePools = append(updatedNodePools, &model.AmazonNodePoolsModel{
				ID:               currentNodePoolMap[nodePoolName].ID,
				CreatedBy:        currentNodePoolMap[nodePoolName].CreatedBy,
				CreatedAt:        currentNodePoolMap[nodePoolName].CreatedAt,
				ClusterID:        currentNodePoolMap[nodePoolName].ClusterID,
				Name:             nodePoolName,
				NodeInstanceType: currentNodePoolMap[nodePoolName].NodeInstanceType,
				NodeImage:        currentNodePoolMap[nodePoolName].NodeImage,
				NodeSpotPrice:    currentNodePoolMap[nodePoolName].NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.MinCount,
				NodeMaxCount:     nodePool.MaxCount,
				Count:            nodePool.Count,
				Delete:           false,
			})

		} else {
			// new node pool

			// ---- [ Node instanceType check ] ---- //
			if len(nodePool.InstanceType) == 0 {
				c.log.Errorf("instanceType is missing for nodePool %v", nodePoolName)
				return nil, pkgErrors.ErrorInstancetypeFieldIsEmpty
			}

			// ---- [ Node image check ] ---- //
			if len(nodePool.Image) == 0 {
				c.log.Errorf("image is missing for nodePool %v", nodePoolName)
				return nil, pkgErrors.ErrorAmazonImageFieldIsEmpty
			}

			// ---- [ Node spot price ] ---- //
			if len(nodePool.SpotPrice) == 0 {
				nodePool.SpotPrice = pkgEks.DefaultSpotPrice
			}

			updatedNodePools = append(updatedNodePools, &model.AmazonNodePoolsModel{
				CreatedBy:        userId,
				Name:             nodePoolName,
				NodeInstanceType: nodePool.InstanceType,
				NodeImage:        nodePool.Image,
				NodeSpotPrice:    nodePool.SpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.MinCount,
				NodeMaxCount:     nodePool.MaxCount,
				Count:            nodePool.Count,
				Delete:           false,
			})
		}
	}

	for _, nodePool := range c.modelCluster.EKS.NodePools {
		if requestedNodePools[nodePool.Name] == nil {
			updatedNodePools = append(updatedNodePools, &model.AmazonNodePoolsModel{
				ID:        nodePool.ID,
				ClusterID: nodePool.ClusterID,
				Name:      nodePool.Name,
				Labels:    nodePool.Labels,
				CreatedAt: nodePool.CreatedAt,
				Delete:    true,
			})
		}
	}
	return updatedNodePools, nil
}

// UpdateCluster updates EKS cluster in cloud
func (c *EKSCluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest, updatedBy uint) error {
	c.log.Info("Start updating EKS cluster")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return err
	}

	var actions []utils.Action

	clusterStackName := c.generateStackNameForCluster()
	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(clusterStackName)}
	cloudformationSrv := cloudformation.New(session)
	autoscalingSrv := autoscaling.New(session)
	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil
	}

	var vpcId, subnetIds, securityGroupId, nodeSecurityGroupId, nodeInstanceRoleId, clusterUserArn, clusterUserAccessKeyId, clusterUserSecretAccessKey string
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "SecurityGroups":
			securityGroupId = aws.StringValue(output.OutputValue)
		case "NodeSecurityGroup":
			nodeSecurityGroupId = aws.StringValue(output.OutputValue)
		case "VpcId":
			vpcId = aws.StringValue(output.OutputValue)
		case "SubnetIds":
			subnetIds = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleId":
			nodeInstanceRoleId = aws.StringValue(output.OutputValue)
		case "ClusterUserArn":
			clusterUserArn = aws.StringValue(output.OutputValue)
		}
	}

	clusterUserAccessKeyId, clusterUserSecretAccessKey, err = action.GetClusterUserAccessKeyIdAndSecretVault(c.GetOrganizationId(), c.GetName())
	if err != nil {
		return err
	}

	if len(securityGroupId) == 0 {
		return errors.New("securityGroupId output not found on stack: " + clusterStackName)
	}
	if len(vpcId) == 0 {
		return errors.New("vpcId output not found on stack: " + clusterStackName)
	}
	if len(subnetIds) == 0 {
		return errors.New("subnetIds output not found on stack: " + clusterStackName)
	}

	nodePoolTemplate, err := pkgEks.GetNodePoolTemplate()
	if err != nil {
		log.Errorln("Getting CloudFormation template for node pools failed: ", err.Error())
		return err
	}

	modelNodePools, err := c.createNodePoolsFromUpdateRequest(updateRequest.EKS.NodePools, updatedBy)
	if err != nil {
		return err
	}

	createUpdateContext := action.NewEksClusterUpdateContext(
		session,
		c.modelCluster.Name,
		aws.String(securityGroupId),
		aws.String(nodeSecurityGroupId),
		aws.StringSlice(strings.Split(subnetIds, ",")),
		c.generateSSHKeyNameForCluster(),
		nodePoolTemplate,
		aws.String(vpcId),
		aws.String(nodeInstanceRoleId),
		clusterUserArn,
		clusterUserAccessKeyId,
		clusterUserSecretAccessKey,
	)
	createUpdateContext.ScaleEnabled = c.GetScaleOptions() != nil && c.GetScaleOptions().Enabled

	deleteContext := action.NewEksClusterDeleteContext(
		session,
		c.modelCluster.Name,
	)

	var nodePoolsToCreate []*model.AmazonNodePoolsModel
	var nodePoolsToUpdate []*model.AmazonNodePoolsModel
	var nodePoolsToDelete []string

	for _, nodePool := range modelNodePools {

		stackName := c.generateNodePoolStackName(nodePool)
		describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
		describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
		if err == nil {
			// delete nodePool
			if nodePool.Delete {
				c.log.Infof("nodePool %v will be deleted", nodePool.Name)
				nodePoolsToDelete = append(nodePoolsToDelete, c.generateNodePoolStackName(nodePool))
				continue
			}
			// update nodePool
			c.log.Infof("nodePool %v already exists will be updated", nodePool.Name)
			// load params which are not updatable from nodeGroup Stack
			for _, param := range describeStacksOutput.Stacks[0].Parameters {
				switch *param.ParameterKey {
				case "NodeImageId":
					nodePool.NodeImage = *param.ParameterValue
				case "NodeInstanceType":
					nodePool.NodeInstanceType = *param.ParameterValue
				case "NodeSpotPrice":
					nodePool.NodeSpotPrice = *param.ParameterValue
				}
			}
			// get current Desired count from ASG linked to nodeGroup stack if Autoscaling is enabled, as we don't to override
			// in this case only min/max counts
			group, err := getAutoScalingGroup(cloudformationSrv, autoscalingSrv, stackName)
			if err != nil {
				c.log.Errorf("unable to find ASG for stack: %v", stackName)
				return err
			}

			// override nodePool.Count with current DesiredCapacity in case of autoscale, as we don't want allow direct
			// setting of DesiredCapacity via API, however we have to limit it to be between new min, max values.
			if nodePool.Autoscaling {
				if group.DesiredCapacity != nil {
					nodePool.Count = int(*group.DesiredCapacity)
				}
				if nodePool.Count < nodePool.NodeMinCount {
					nodePool.Count = nodePool.NodeMinCount
				}
				if nodePool.Count > nodePool.NodeMaxCount {
					nodePool.Count = nodePool.NodeMaxCount
				}
				c.log.Infof("DesiredCapacity for %v will be: %v", *group.AutoScalingGroupARN, nodePool.Count)
			}

			nodePoolsToUpdate = append(nodePoolsToUpdate, nodePool)
		} else {
			if nodePool.Delete {
				c.log.Warnf("nodePool %v to be deleted doesn't exists: %v", nodePool.Name, err)
				continue
			}
			// create nodePool
			c.log.Infof("nodePool %v doesn't exists will be created", nodePool.Name)
			nodePoolsToCreate = append(nodePoolsToCreate, nodePool)
		}
	}

	ASGWaitLoopCount := int(viper.GetDuration(config.EksASGFulfillmentTimeout).Seconds() / asgWaitLoopSleepSeconds)
	headNodePoolName := viper.GetString(config.PipelineHeadNodePoolName)

	deleteNodePoolAction := action.NewDeleteStacksAction(c.log, deleteContext, nodePoolsToDelete...)
	createNodePoolAction := action.NewCreateUpdateNodePoolStackAction(c.log, true, createUpdateContext, ASGWaitLoopCount, asgWaitLoopSleepSeconds*time.Second, headNodePoolName, nodePoolsToCreate...)
	updateNodePoolAction := action.NewCreateUpdateNodePoolStackAction(c.log, false, createUpdateContext, ASGWaitLoopCount, asgWaitLoopSleepSeconds*time.Second, headNodePoolName, nodePoolsToUpdate...)

	actions = append(actions, createNodePoolAction, updateNodePoolAction, deleteNodePoolAction)

	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, false)
	if err != nil {
		c.log.Errorln("EKS cluster update error:", err.Error())
		return err
	}

	c.modelCluster.EKS.NodePools = modelNodePools

	return nil
}

// UpdateNodePools updates nodes pools of a cluster
func (c *EKSCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	c.log.Info("Start updating nodepools")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return emperror.Wrap(err, "Error retrieving AWS credentials")
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return emperror.Wrap(err, "Error creating AWS session")
	}

	autoscalingSrv := autoscaling.New(session)
	cloudformationSrv := cloudformation.New(session)

	waitRoutines := 0
	waitChan := make(chan error)
	defer close(waitChan)

	caughtErrors := emperror.NewMultiErrorBuilder()
	ASGWaitLoopCount := int(viper.GetDuration(config.EksASGFulfillmentTimeout).Seconds() / asgWaitLoopSleepSeconds)

	for poolName, nodePool := range request.NodePools {

		asgName, err := c.getAutoScalingGroupName(cloudformationSrv, autoscalingSrv, poolName)
		if err != nil {
			c.log.Errorf("Error ASG not found for node pool %v. %v", poolName, err.Error())
			continue
		}
		params := &autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(*asgName),
			DesiredCapacity:      aws.Int64(int64(nodePool.Count)),
			HonorCooldown:        aws.Bool(false),
		}
		c.log.Infof("Setting node pool %s size to %d", poolName, nodePool.Count)
		_, err = autoscalingSrv.SetDesiredCapacity(params)
		if err != nil {
			c.log.Errorf("Error setting node pool %s size: %v", poolName, err)
			caughtErrors.Add(err)
			continue
		}
		c.setNodePoolSize(poolName, nodePool.Count)

		waitRoutines++
		go func(poolName string) {
			waitChan <- action.WaitForASGToBeFulfilled(session, c.log, c.modelCluster.Name,
				poolName, ASGWaitLoopCount, asgWaitLoopSleepSeconds*time.Second)
		}(poolName)

	}

	// wait for goroutines to finish
	for i := 0; i < waitRoutines; i++ {
		waitErr := <-waitChan
		if waitErr != nil {
			caughtErrors.Add(waitErr)
		}
	}

	return caughtErrors.ErrOrNil()
}

func getAutoScalingGroup(cloudformationSrv *cloudformation.CloudFormation, autoscalingSrv *autoscaling.AutoScaling, stackName string) (*autoscaling.Group, error) {
	logResourceId := "NodeGroup"
	describeStackResourceInput := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: &logResourceId,
		StackName:         aws.String(stackName)}
	describeStacksOutput, err := cloudformationSrv.DescribeStackResource(describeStackResourceInput)
	if err != nil {
		return nil, err
	}

	describeAutoScalingGroupsInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			describeStacksOutput.StackResourceDetail.PhysicalResourceId,
		},
	}
	describeAutoScalingGroupsOutput, err := autoscalingSrv.DescribeAutoScalingGroups(&describeAutoScalingGroupsInput)
	if err != nil {
		return nil, err
	}

	return describeAutoScalingGroupsOutput.AutoScalingGroups[0], nil
}

func (c *EKSCluster) getAutoScalingGroupName(cloudformationSrv *cloudformation.CloudFormation, autoscalingSrv *autoscaling.AutoScaling, nodePoolName string) (*string, error) {
	logResourceId := "NodeGroup"
	stackName := action.GenerateNodePoolStackName(c.modelCluster.Name, nodePoolName)
	describeStackResourceInput := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: &logResourceId,
		StackName:         aws.String(stackName)}
	describeStacksOutput, err := cloudformationSrv.DescribeStackResource(describeStackResourceInput)
	if err != nil {
		return nil, err
	}

	return describeStacksOutput.StackResourceDetail.PhysicalResourceId, nil
}

// GenerateK8sConfig generates kube config for this EKS cluster which authenticates through the aws-iam-authenticator,
// you have to install with: go get github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
func (c *EKSCluster) GenerateK8sConfig() *clientcmdapi.Config {
	return &clientcmdapi.Config{
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: c.modelCluster.Name,
				Cluster: clientcmdapi.Cluster{
					Server:                   c.APIEndpoint,
					CertificateAuthorityData: c.CertificateAuthorityData,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: c.modelCluster.Name,
				Context: clientcmdapi.Context{
					AuthInfo: "eks",
					Cluster:  c.modelCluster.Name,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: "eks",
				AuthInfo: clientcmdapi.AuthInfo{
					Exec: &clientcmdapi.ExecConfig{
						APIVersion: "client.authentication.k8s.io/v1alpha1",
						Command:    "aws-iam-authenticator",
						Args:       []string{"token", "-i", c.modelCluster.Name},
						Env: []clientcmdapi.ExecEnvVar{
							{Name: "AWS_ACCESS_KEY_ID", Value: c.awsAccessKeyID},
							{Name: "AWS_SECRET_ACCESS_KEY", Value: c.awsSecretAccessKey},
						},
					},
				},
			},
		},
		Kind:           "Config",
		CurrentContext: c.modelCluster.Name,
	}
}

// DownloadK8sConfig generates and marshalls the kube config for this cluster.
func (c *EKSCluster) DownloadK8sConfig() ([]byte, error) {
	if c.APIEndpoint == "" || c.CertificateAuthorityData == nil || c.awsAccessKeyID == "" || c.awsSecretAccessKey == "" {

		awsCred, err := c.createAWSCredentialsFromSecret()
		if err != nil {
			return nil, err
		}

		session, err := session.NewSession(&aws.Config{
			Region:      aws.String(c.modelCluster.Location),
			Credentials: awsCred,
		})
		if err != nil {
			return nil, err
		}

		context := action.NewEksClusterCreationContext(session, c.modelCluster.Name, "", "")

		if err := c.loadEksMasterSettings(context); err != nil {
			return nil, err
		}

		if err := c.loadClusterUserCredentials(context); err != nil {
			return nil, err
		}
	}

	config := c.GenerateK8sConfig()
	return yaml.Marshal(config)
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
		Logging:           c.GetLogging(),
		Monitoring:        c.GetMonitoring(),
		ServiceMesh:       c.GetServiceMesh(),
		SecurityScan:      c.GetSecurityScan(),
		NodePools:         nodePools,
		Version:           c.modelCluster.EKS.Version,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Region:            c.modelCluster.Location,
		TtlMinutes:        c.modelCluster.TtlMinutes,
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

// ListNodeNames returns node names to label them
func (c *EKSCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	// nodes are labeled in create request
	return
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

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return false, err
	}

	eksSvc := eks.New(session)
	describeCluster := &eks.DescribeClusterInput{Name: aws.String(c.GetName())}
	clusterDesc, err := eksSvc.DescribeCluster(describeCluster)
	if err != nil {
		return false, err
	}

	return aws.StringValue(clusterDesc.Cluster.Status) == eks.ClusterStatusActive, nil
}

// ValidateCreationFields validates all fields
func (c *EKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	regions, err := c.CloudInfoClient.GetServiceRegions(pkgCluster.Amazon, pkgCluster.EKS)
	if err != nil {
		return emperror.Wrap(err, "failed to list regions where EKS service is enabled")
	}

	regionFound := false
	for _, region := range regions {
		if region == r.Location {
			regionFound = true
			break
		}
	}

	if !regionFound {
		return pkgErrors.ErrorNotValidLocation
	}

	image, err := ListEksImages(r.Properties.CreateClusterEKS.Version, r.Location)
	if err != nil {
		return emperror.Wrap(err, "failed to get EKS AMI")
	}

	for name, nodePool := range r.Properties.CreateClusterEKS.NodePools {
		if image != nodePool.Image {
			return emperror.With(pkgErrors.ErrorNotValidNodeImage, "image", nodePool.Image, "nodePool", name, "region", r.Location)
		}
	}

	// validate VPC
	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return emperror.Wrap(err, "failed to get cluster AWS credentials")
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return emperror.Wrap(err, "failed to create AWS session")
	}

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(session), NewLogurLogger(c.log))
	if r.Properties.CreateClusterEKS.Vpc != nil {

		if r.Properties.CreateClusterEKS.Vpc.VpcId != "" && r.Properties.CreateClusterEKS.Vpc.Cidr != "" {
			return errors.New("specifying both CIDR and ID for VPC is not allowed")
		}

		if r.Properties.CreateClusterEKS.Vpc.VpcId == "" && r.Properties.CreateClusterEKS.Vpc.Cidr == "" {
			return errors.New("either CIDR or ID is required for VPC")
		}

		if r.Properties.CreateClusterEKS.Vpc.VpcId != "" {
			// verify that the provided VPC exists and is in available state
			exists, err := netSvc.VpcAvailable(r.Properties.CreateClusterEKS.Vpc.VpcId)

			if err != nil {
				return emperror.WrapWith(err, "failed to check if VPC is available", "vpcId", r.Properties.CreateClusterEKS.Vpc.VpcId)
			}

			if !exists {
				return errors.New("VPC not found or it's not in 'available' state")
			}
		}
	}

	// subnets
	if len(r.Properties.CreateClusterEKS.Subnets) != 2 {
		return errors.New("2 Subnets in 2 different AZs are required")
	}

	subnetCidrUsed := r.Properties.CreateClusterEKS.Subnets[0].Cidr != ""

	if !subnetCidrUsed && r.Properties.CreateClusterEKS.Vpc.Cidr != "" {
		return errors.New("if Subnet ID is specified than VPC ID must be provided as well")
	}

	for _, subnet := range r.Properties.CreateClusterEKS.Subnets {
		if subnet.Cidr != "" && subnet.SubnetId != "" {
			return errors.New("specifying both CIDR and ID for a Subnet is not allowed")
		}

		if subnet.Cidr == "" && subnet.SubnetId == "" {
			return errors.New("either CIDR or ID is required for Subnet")
		}

		if subnetCidrUsed {
			if subnet.SubnetId != "" && subnet.Cidr == "" {
				return errors.New("specify either CIDR or ID for all Subnets")
			}
		} else {
			if subnet.SubnetId == "" && subnet.Cidr != "" {
				return errors.New("specify either CIDR or ID for all Subnets")
			}
		}

		if subnet.SubnetId != "" {
			exists, err := netSvc.SubnetAvailable(subnet.SubnetId, r.Properties.CreateClusterEKS.Vpc.VpcId)
			if err != nil {
				return emperror.WrapWith(err, "failed to check if Subnet is available in VPC")
			}
			if !exists {
				return fmt.Errorf("subnet '%s' not found in VPC or it's not in 'available' state", subnet.SubnetId)
			}
		}
	}

	// route table
	// if VPC ID and Subnet CIDR is provided than Route Table ID is required as well.

	if r.Properties.CreateClusterEKS.Vpc.VpcId != "" && subnetCidrUsed {
		if r.Properties.CreateClusterEKS.RouteTableId == "" {
			return errors.New("if VPC ID specified and CIDR for Subnets, Route Table ID must be provided as well")
		}

		// verify if provided route table exists
		exists, err := netSvc.RouteTableAvailable(r.Properties.CreateClusterEKS.RouteTableId, r.Properties.CreateClusterEKS.Vpc.VpcId)
		if err != nil {
			return emperror.WrapWith(err, "failed to check if RouteTable is available",
				"vpcId", r.Properties.CreateClusterEKS.Vpc.VpcId,
				"routeTableId", r.Properties.CreateClusterEKS.RouteTableId)
		}
		if !exists {
			return errors.New("Route Table not found in the given VPC or it's not in 'active' state")
		}

	} else {
		if r.Properties.CreateClusterEKS.RouteTableId != "" {
			return errors.New("Route Table ID should be provided only when VPC ID and CIDR for Subnets are specified")
		}
	}

	return nil
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

// GetK8sIpv4Cidrs returns possible IP ranges for pods and services in the cluster
// On EKS the services IP range is chosen from two possible ranges
// source: https://forums.aws.amazon.com/thread.jspa?messageID=859958
// On EKS the pods IP range is coming from the CIDR's of the subnets
func (c *EKSCluster) GetK8sIpv4Cidrs() (*pkgCluster.Ipv4Cidrs, error) {
	eksServiceClusterIPRanges := []string{"172.20.0.0/16", "10.100.0.0/16"}

	var eksPodIPRanges []string
	for _, subnet := range c.GetModel().EKS.Subnets {
		eksPodIPRanges = append(eksPodIPRanges, *subnet.Cidr)
	}

	return &pkgCluster.Ipv4Cidrs{
		ServiceClusterIPRanges: eksServiceClusterIPRanges,
		PodIPRanges:            eksPodIPRanges,
	}, nil
}

// GetK8sConfig returns the Kubernetes config
func (c *EKSCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *EKSCluster) RequiresSshPublicKey() bool {
	return true
}

// ListEksImages returns AMIs for EKS
func ListEksImages(version, region string) (string, error) {
	// TODO: revise this once CloudInfo can provide the correct EKS AMIs dynamically at runtime
	ami, err := pkgEks.GetDefaultImageID(region, version)
	if err != nil {
		return "", emperror.Wrapf(err, "couldn't get EKS AMI for Kubernetes version %q in region %q", version, region)
	}

	return ami, nil
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *EKSCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetSecurityScan returns true if security scan enabled on the cluster
func (c *EKSCluster) GetSecurityScan() bool {
	return c.modelCluster.SecurityScan
}

// SetSecurityScan returns true if security scan enabled on the cluster
func (c *EKSCluster) SetSecurityScan(scan bool) {
	c.modelCluster.SecurityScan = scan
}

// GetLogging returns true if logging enabled on the cluster
func (c *EKSCluster) GetLogging() bool {
	return c.modelCluster.Logging
}

// SetLogging returns true if logging enabled on the cluster
func (c *EKSCluster) SetLogging(l bool) {
	c.modelCluster.Logging = l
}

// GetMonitoring returns true if momnitoring enabled on the cluster
func (c *EKSCluster) GetMonitoring() bool {
	return c.modelCluster.Monitoring
}

// SetMonitoring returns true if monitoring enabled on the cluster
func (c *EKSCluster) SetMonitoring(l bool) {
	c.modelCluster.Monitoring = l
}

// GetScaleOptions returns scale options for the cluster
func (c *EKSCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.modelCluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *EKSCluster) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.modelCluster.ScaleOptions, scaleOptions)
}

// GetServiceMesh returns true if service mesh is enabled on the cluster
func (c *EKSCluster) GetServiceMesh() bool {
	return c.modelCluster.ServiceMesh
}

// SetServiceMesh sets service mesh flag on the cluster
func (c *EKSCluster) SetServiceMesh(m bool) {
	c.modelCluster.ServiceMesh = m
}

// GetTTL retrieves the TTL of the cluster
func (c *EKSCluster) GetTTL() time.Duration {
	return time.Duration(c.modelCluster.TtlMinutes) * time.Minute
}

// SetTTL sets the lifespan of a cluster
func (c *EKSCluster) SetTTL(ttl time.Duration) {
	c.modelCluster.TtlMinutes = uint(ttl.Minutes())
}

// GetEKSNodePools returns EKS node pools from a common cluster.
func GetEKSNodePools(cluster CommonCluster) ([]*model.AmazonNodePoolsModel, error) {
	ekscluster, ok := cluster.(*EKSCluster)
	if !ok {
		return nil, ErrInvalidClusterInstance
	}

	return ekscluster.modelCluster.EKS.NodePools, nil
}

// loadEksMasterSettings gets K8s API server endpoint and Certificate Authority data from AWS and populates into
// this EKSCluster instance
func (c *EKSCluster) loadEksMasterSettings(context *action.EksClusterCreateUpdateContext) error {
	if c.APIEndpoint == "" || c.CertificateAuthorityData == nil {
		// Get cluster API endpoint and cluster CA data
		loadEksSettings := action.NewLoadEksSettingsAction(c.log, context)
		_, err := loadEksSettings.ExecuteAction(nil)
		if err != nil {
			return err
		}

		c.APIEndpoint = aws.StringValue(context.APIEndpoint)
		c.CertificateAuthorityData, err = base64.StdEncoding.DecodeString(aws.StringValue(context.CertificateAuthorityData))
		if err != nil {
			return err
		}
	}

	return nil
}

// loadClusterUserCredentials get the cluster user credentials from AWS and populates into this EKSCluster instance
func (c *EKSCluster) loadClusterUserCredentials(context *action.EksClusterCreateUpdateContext) error {
	// Get IAM user access key id and secret
	if c.awsAccessKeyID == "" || c.awsSecretAccessKey == "" {

		clusterUserAccessKeyId, clusterUserSecretAccessKey, err := action.GetClusterUserAccessKeyIdAndSecretVault(c.GetOrganizationId(), context.ClusterName)

		if err != nil {
			return emperror.Wrap(err, "getting user access key and secret failed")
		}

		context.ClusterUserAccessKeyId = clusterUserAccessKeyId
		context.ClusterUserSecretAccessKey = clusterUserSecretAccessKey

		c.awsAccessKeyID = clusterUserAccessKeyId
		c.awsSecretAccessKey = clusterUserSecretAccessKey
	}

	return nil
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *EKSCluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *EKSCluster) GetKubernetesUserName() (string, error) {
	return "", nil
}

// GetCreatedBy returns cluster create userID.
func (c *EKSCluster) GetCreatedBy() uint {
	return c.modelCluster.CreatedBy
}
