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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks/action"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
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

//CreateEKSClusterFromRequest creates ClusterModel struct from the request
func CreateEKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EKSCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
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
			Version:   request.Properties.CreateClusterEKS.Version,
			NodePools: modelNodePools,
		},
		CreatedBy: userId,
	}
	return &cluster, nil
}

//EKSCluster struct for EKS cluster
type EKSCluster struct {
	modelCluster             *model.ClusterModel
	APIEndpoint              string
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

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *EKSCluster) GetAPIEndpoint() (string, error) {
	return c.APIEndpoint, nil
}

//CreateEKSClusterFromModel creates ClusterModel struct from the model
func CreateEKSClusterFromModel(clusterModel *model.ClusterModel) (*EKSCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	eksCluster := EKSCluster{
		modelCluster: clusterModel,
		log:          log.WithField("cluster", clusterModel.Name),
	}
	return &eksCluster, nil
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
		return err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return err
	}

	// role that controls access to resources for creating an EKS cluster
	eksStackName := c.generateStackNameForCluster()
	sshKeyName := c.generateSSHKeyNameForCluster()

	c.modelCluster.RbacEnabled = true

	log.Infoln("Getting CloudFormation template for creating node pools for EKS cluster")
	nodePoolTemplate, err := pkgEks.GetNodePoolTemplate()
	if err != nil {
		log.Errorln("Getting CloudFormation template for node pools failed: ", err.Error())
		return err
	}

	creationContext := action.NewEksClusterCreationContext(
		session,
		c.modelCluster.Name,
		sshKeyName,
		nodePoolTemplate,
	)

	sshSecret, err := c.getSshSecret(c)
	if err != nil {
		return err
	}

	actions := []utils.Action{
		action.NewCreateVPCAndRolesAction(c.log, creationContext, eksStackName),
		action.NewCreateClusterUserAccessKeyAction(c.log, creationContext),
		action.NewPersistClusterUserAccessKeyAction(c.log, creationContext, c.GetOrganizationId()),
		action.NewUploadSSHKeyAction(c.log, creationContext, sshSecret),
		action.NewGenerateVPCConfigRequestAction(c.log, creationContext, eksStackName, c.GetOrganizationId()),
		action.NewCreateEksClusterAction(c.log, creationContext, c.modelCluster.EKS.Version),
		action.NewCreateUpdateNodePoolStackAction(c.log, true, creationContext, c.modelCluster.EKS.NodePools...),
	}

	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, true)
	if err != nil {
		c.log.Errorln("EKS cluster create error:", err.Error())
		return err
	}

	c.APIEndpoint = aws.StringValue(creationContext.APIEndpoint)
	c.CertificateAuthorityData, err = base64.StdEncoding.DecodeString(aws.StringValue(creationContext.CertificateAuthorityData))

	if err != nil {
		c.log.Errorf("Decoding base64 format EKS K8S certificate authority data failed: %s", err.Error())
		return err
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
		return err
	}

	restKubeConfig, err := helm.GetK8sClientConfig(kubeConfig)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(restKubeConfig)
	if err != nil {
		return err
	}

	// create default storage class
	err = createDefaultStorageClass(kubeClient, "kubernetes.io/aws-ebs")
	if err != nil {
		return err
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
		return err
	}

	err = c.modelCluster.Save()
	if err != nil {
		return err
	}

	c.log.Info("EKS cluster created.")

	return nil
}

func (c *EKSCluster) generateSSHKeyNameForCluster() string {
	return c.modelCluster.Name + "-pipeline-eks-ssh"
}

func (c *EKSCluster) generateNodePoolStackName(nodePool *model.AmazonNodePoolsModel) string {
	return c.modelCluster.Name + "-pipeline-eks-nodepool-" + nodePool.Name
}

func (c *EKSCluster) generateStackNameForCluster() string {
	return c.modelCluster.Name + "-pipeline-eks"
}

func (c *EKSCluster) generateIAMRoleNameForCluster() string {
	return c.modelCluster.Name + "-pipeline-eks"
}

// Persist saves the cluster model
func (c *EKSCluster) Persist(status, statusMessage string) error {
	c.log.Infof("Model before save: %v", c.modelCluster)
	return c.modelCluster.UpdateStatus(status, statusMessage)
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

	nodePoolStacks := make([]string, 0, len(c.modelCluster.EKS.NodePools))
	for _, nodePool := range c.modelCluster.EKS.NodePools {
		nodePoolStackName := c.generateNodePoolStackName(nodePool)
		nodePoolStacks = append(nodePoolStacks, nodePoolStackName)
	}
	deleteNodePoolsAction := action.NewDeleteStacksAction(c.log, deleteContext, nodePoolStacks...)

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

func (c *EKSCluster) createNodePoolsFromUpdateRequest(requestedNodePools map[string]*ec2.NodePool, userId uint) ([]*model.AmazonNodePoolsModel, error) {

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
				NodeInstanceType: nodePool.InstanceType,
				NodeImage:        nodePool.Image,
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
				nodePool.SpotPrice = ec2.DefaultSpotPrice
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
				CreatedBy: nodePool.CreatedBy,
				CreatedAt: nodePool.CreatedAt,
				ClusterID: nodePool.ClusterID,
				Name:      nodePool.Name,
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

	deleteNodePoolAction := action.NewDeleteStacksAction(c.log, deleteContext, nodePoolsToDelete...)
	createNodePoolAction := action.NewCreateUpdateNodePoolStackAction(c.log, true, createUpdateContext, nodePoolsToCreate...)
	updateNodePoolAction := action.NewCreateUpdateNodePoolStackAction(c.log, false, createUpdateContext, nodePoolsToUpdate...)

	actions = append(actions, deleteNodePoolAction, createNodePoolAction, updateNodePoolAction)

	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, false)
	if err != nil {
		c.log.Errorln("EKS cluster update error:", err.Error())
		return err
	}

	c.modelCluster.EKS.NodePools = modelNodePools

	return nil
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

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.EKS.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:  np.Autoscaling,
				Count:        np.Count,
				InstanceType: np.NodeInstanceType,
				SpotPrice:    np.NodeSpotPrice,
				MinCount:     np.NodeMinCount,
				MaxCount:     np.NodeMaxCount,
				Image:        np.NodeImage,
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
		NodePools:         nodePools,
		Version:           c.modelCluster.EKS.Version,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
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
	return CheckEqualityToUpdate(r, c.modelCluster.EKS.NodePools)
}

// AddDefaultsToUpdate adds defaults to update request
func (c *EKSCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {
	defaultImage := pkgEks.DefaultImages[c.modelCluster.Location]

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

// UpdateStatus updates cluster status in database
func (c *EKSCluster) UpdateStatus(status string, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (c *EKSCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	c.log.Infoln("Getting cluster details")

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

	eksSvc := eks.New(session)
	describeCluster := &eks.DescribeClusterInput{Name: aws.String(c.GetName())}
	clusterDesc, err := eksSvc.DescribeCluster(describeCluster)
	if err != nil {
		return nil, err
	}

	nodePools := make(map[string]*pkgCluster.NodeDetails)
	for _, np := range c.modelCluster.EKS.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodeDetails{
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				Version:           aws.StringValue(clusterDesc.Cluster.Version),
				Count:             np.Count,
				MinCount:          np.NodeMinCount,
				MaxCount:          np.NodeMaxCount,
			}
		}
	}

	if aws.StringValue(clusterDesc.Cluster.Status) == eks.ClusterStatusActive {
		return &pkgCluster.DetailsResponse{
			CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
			Name:              c.modelCluster.Name,
			Id:                c.modelCluster.ID,
			Location:          c.modelCluster.Location,
			MasterVersion:     aws.StringValue(clusterDesc.Cluster.Version),
			NodePools:         nodePools,
			Endpoint:          c.APIEndpoint,
			Status:            c.modelCluster.Status,
		}, nil
	}

	return nil, pkgErrors.ErrorClusterNotReady
}

// ValidateCreationFields validates all fields
func (c *EKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	regions, err := ListEksRegions(c.GetOrganizationId(), c.GetSecretId())
	if err != nil {
		c.log.Errorf("Listing regions where EKS service is available failed: %s", err.Error())
		return err
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

	imagesInRegion, err := ListEksImages(r.Location)
	if err != nil {
		c.log.Errorf("Listing AMIs that that support EKS failed: %s", err.Error())
		return err
	}

	for name, nodePool := range r.Properties.CreateClusterEKS.NodePools {
		images, ok := imagesInRegion[r.Location]
		if !ok {
			c.log.Errorf("Image %q provided for node pool %q is not valid", name, nodePool.Image)
			return pkgErrors.ErrorNotValidNodeImage
		}

		for _, image := range images {
			if image != nodePool.Image {
				c.log.Errorf("Image %q provided for node pool %q is not valid", name, nodePool.Image)
				return pkgErrors.ErrorNotValidNodeImage
			}
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

// GetConfigSecretId return config secret id
func (c *EKSCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
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

// ListEksRegions returns the regions in which AmazonEKS service is enabled
func ListEksRegions(orgId uint, secretId string) ([]string, error) {
	// AWS API https://docs.aws.amazon.com/sdk-for-go/api/aws/endpoints/ doesn't recognizes AmazonEKS service yet
	// thus we can not use it to query what locations the service is enabled in.

	// We'll use the pricing API to determine what locations the service is enabled in.

	// TODO revisit this later when https://docs.aws.amazon.com/sdk-for-go/api/aws/endpoints/ starts supporting AmazonEKS

	secret, err := secret.Store.Get(orgId, secretId)
	if err != nil {
		return nil, err
	}

	credentials := verify.CreateAWSCredentials(secret.Values)
	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(pkgEks.UsEast1), // pricing API available in us-east-1
		Credentials: credentials,
	})
	if err != nil {
		return nil, err
	}

	svc := pricing.New(session)

	getAttrValuesInput := &pricing.GetAttributeValuesInput{
		AttributeName: aws.String(pkgCluster.KeyWordLocation),
		ServiceCode:   aws.String("AmazonEKS"),
	}
	attributeValues, err := svc.GetAttributeValues(getAttrValuesInput)
	if err != nil {
		return nil, err
	}

	var eksLocations []string
	for _, attrValue := range attributeValues.AttributeValues {
		eksLocations = append(eksLocations, aws.StringValue(attrValue.Value))
	}

	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	var eksRegionIds []string
	for _, p := range partitions {
		for _, r := range p.Regions() {
			for _, eksLocation := range eksLocations {
				if r.Description() == eksLocation {
					eksRegionIds = append(eksRegionIds, r.ID())
				}
			}
		}

	}

	return eksRegionIds, nil
}

// ListEksImages returns AMIs for EKS
func ListEksImages(region string) (map[string][]string, error) {
	// currently there are only two AMIs for EKS.
	// TODO: revise this once there is AWS API for retrieving EKS AMIs dynamically at runtime
	ami, ok := pkgEks.DefaultImages[region]
	if ok {
		return map[string][]string{
			region: {ami},
		}, nil
	}

	return map[string][]string{
		region: {},
	}, nil
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *EKSCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
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
		eksStackName := c.generateStackNameForCluster()
		getVPCConfig := action.NewGenerateVPCConfigRequestAction(c.log, context, eksStackName, c.GetOrganizationId())

		_, err := getVPCConfig.ExecuteAction(nil)
		if err != nil {
			return err
		}

		c.awsAccessKeyID = context.ClusterUserAccessKeyId
		c.awsSecretAccessKey = context.ClusterUserSecretAccessKey
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
