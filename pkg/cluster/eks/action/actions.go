package action

import (
	"fmt"
	"time"

	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

// --

// EksClusterCreationContext describes the properties of an EKS cluster creation
type EksClusterCreationContext struct {
	Session                  *session.Session
	ClusterName              string
	NodeInstanceRoles        []string
	Role                     *iam.Role
	SecurityGroupID          *string
	SubnetIDs                []*string
	SSHKeyName               string
	SSHKey                   *secret.SSHKeyPair
	VpcID                    *string
	ProvidedRoleArn          string
	APIEndpoint              *string
	CertificateAuthorityData *string
}

// NewEksClusterCreationContext creates a new EksClusterCreationContext
func NewEksClusterCreationContext(session *session.Session, clusterName string, sshKeyName string) *EksClusterCreationContext {
	return &EksClusterCreationContext{
		Session:     session,
		ClusterName: clusterName,
		SSHKeyName:  sshKeyName,
	}
}

// EksClusterDeletionContext describes the properties of an EKS cluster deletion
type EksClusterDeletionContext struct {
	Session     *session.Session
	ClusterName string
}

// NewEksClusterDeleteContext creates a new NewEksClusterDeleteContext
func NewEksClusterDeleteContext(session *session.Session, clusterName string) *EksClusterDeletionContext {
	return &EksClusterDeletionContext{
		Session:     session,
		ClusterName: clusterName,
	}
}

// --

var _ utils.RevocableAction = (*CreateVPCAction)(nil)

// EnsureIAMRoleAction describes how to create an IAM role for EKS
type EnsureIAMRoleAction struct {
	context                   *EksClusterCreationContext
	roleName                  string
	rolesToAttach             []string
	successfullyAttachedRoles []string
}

// NewEnsureIAMRoleAction creates a new NewEnsureIAMRoleAction
func NewEnsureIAMRoleAction(creationContext *EksClusterCreationContext, roleName string) *EnsureIAMRoleAction {
	return &EnsureIAMRoleAction{
		context:  creationContext,
		roleName: roleName,
		rolesToAttach: []string{
			"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
			"arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
		},
		successfullyAttachedRoles: []string{},
	}
}

// GetName returns the name of this EnsureIAMRoleAction
func (action *EnsureIAMRoleAction) GetName() string {
	return "EnsureIAMRoleAction"
}

// ExecuteAction executes this EnsureIAMRoleAction
func (action *EnsureIAMRoleAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE EnsureIAMRoleAction, role name:", action.roleName)

	iamSvc := iam.New(action.context.Session)
	assumeRolePolicy := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "eks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`

	roleinput := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: &assumeRolePolicy,
		RoleName:                 aws.String(action.roleName),
		Description:              aws.String("EKS Creation Role created by Pipeline"),
		Path:                     aws.String("/"),
		MaxSessionDuration:       aws.Int64(3600),
	}
	//irName := ""
	outInstanceRole, err := iamSvc.CreateRole(roleinput)

	if err != nil {
		log.Errorln("CreateRole error:", err.Error())
		return nil, err
	}

	for _, roleName := range action.rolesToAttach {
		attachRoleInput := &iam.AttachRolePolicyInput{
			RoleName:  outInstanceRole.Role.RoleName,
			PolicyArn: aws.String(roleName),
		}
		_, err = iamSvc.AttachRolePolicy(attachRoleInput)
		if err != nil {
			log.Errorln("AttachRole error:", err.Error())
			return nil, err
		}
		action.successfullyAttachedRoles = append(action.successfullyAttachedRoles, roleName)
	}
	action.context.Role = outInstanceRole.Role

	return outInstanceRole.Role, nil
}

// UndoAction rolls back this EnsureIAMRoleAction
func (action *EnsureIAMRoleAction) UndoAction() (err error) {
	log.Infoln("EXECUTE UNDO EnsureIAMRoleAction, deleting role:", action.roleName)

	iamSvc := iam.New(action.context.Session)

	//detach role policies first
	for _, roleName := range action.successfullyAttachedRoles {
		detachRolePolicyInput := &iam.DetachRolePolicyInput{
			RoleName:  action.context.Role.RoleName,
			PolicyArn: aws.String(roleName),
		}
		_, err = iamSvc.DetachRolePolicy(detachRolePolicyInput)
		if err != nil {
			log.Debug("DetachRole error: %v", err)
			return err
		}
	}
	//delete role
	deleteRoleInput := &iam.DeleteRoleInput{
		RoleName: aws.String(action.roleName),
	}
	_, err = iamSvc.DeleteRole(deleteRoleInput)
	return err
}

// --

var _ utils.RevocableAction = (*CreateVPCAction)(nil)

// CreateVPCAction describes the properties of a VPC creation
type CreateVPCAction struct {
	context   *EksClusterCreationContext
	stackName string
	//describeStacksTimeInterval time.Duration
	//stackCreationTimeout       time.Duration
}

// NewCreateVPCAction creates a new CreateVPCAction
func NewCreateVPCAction(creationContext *EksClusterCreationContext, stackName string) *CreateVPCAction {
	return &CreateVPCAction{
		context:   creationContext,
		stackName: stackName,
		//describeStacksTimeInterval: 10 * time.Second,
		//stackCreationTimeout:       3 * time.Minute,
	}
}

// GetName returns the name of this CreateVPCAction
func (action *CreateVPCAction) GetName() string {
	return "CreateVPCAction"
}

// ExecuteAction executes this CreateVPCAction
func (action *CreateVPCAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE CreateVPCAction, stack name:", action.stackName)

	cloudformationSrv := cloudformation.New(action.context.Session)
	createStackInput := &cloudformation.CreateStackInput{
		//Capabilities:       []*string{},
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		DisableRollback:    aws.Bool(false),
		//Parameters:         []*cloudformation.Parameter{
		//&cloudformation.Parameter{
		//ParameterKey:   aws.String("foobar"),
		//ParameterValue: aws.String("foobar2"),
		//},
		//},
		StackName:        aws.String(action.stackName),
		Tags:             []*cloudformation.Tag{{Key: aws.String("pipeline-created"), Value: aws.String("true")}},
		TemplateURL:      aws.String("https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-06-05/amazon-eks-vpc-sample.yaml"),
		TimeoutInMinutes: aws.Int64(10),
	}
	//startTime := time.Now()
	_, err = cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		return
	}

	//action.context.VpcID = createStackOutput.StackId
	//completed := false
	//for time.Now().Before(startTime.Add(action.stackCreationTimeout)) {
	//	stackStatuses, err := cloudformationSrv.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(action.stackName)})
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	if len(stackStatuses.Stacks) != 1 {
	//		return nil, errors.New(fmt.Sprintf("Got %d stack(s) instead of 1 after stack creation", len(stackStatuses.Stacks)))
	//	}
	//	stack := stackStatuses.Stacks[0]
	//	fmt.Printf("stackStatus: %s\n", *stack.StackStatus)
	//	if *stack.StackStatus == cloudformation.StackStatusCreateComplete {
	//		completed = true
	//		break
	//	}
	//	time.Sleep(action.describeStacksTimeInterval)
	//}

	//if !completed {
	//	return nil, errors.New("Timeout occurred during eks stack creation")
	//}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(action.stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)

	return nil, err
}

// UndoAction rolls back this CreateVPCAction
func (action *CreateVPCAction) UndoAction() (err error) {
	log.Infoln("EXECUTE UNDO CreateVPCAction, deleting stack:", action.stackName)
	cloudformationSrv := cloudformation.New(action.context.Session)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		StackName:          aws.String(action.stackName),
	}
	_, err = cloudformationSrv.DeleteStack(deleteStackInput)
	return err
}

// --

var _ utils.RevocableAction = (*GenerateVPCConfigRequestAction)(nil)

// GenerateVPCConfigRequestAction describes how to request a VPC config
type GenerateVPCConfigRequestAction struct {
	context   *EksClusterCreationContext
	stackName string
}

// NewGenerateVPCConfigRequestAction creates a new GenerateVPCConfigRequestAction
func NewGenerateVPCConfigRequestAction(creationContext *EksClusterCreationContext, stackName string) *GenerateVPCConfigRequestAction {
	return &GenerateVPCConfigRequestAction{
		context:   creationContext,
		stackName: stackName,
	}
}

// GetName returns the name of this GenerateVPCConfigRequestAction
func (action *GenerateVPCConfigRequestAction) GetName() string {
	return "GenerateVPCConfigRequestAction"
}

// ExecuteAction executes this GenerateVPCConfigRequestAction
func (action *GenerateVPCConfigRequestAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE GenerateVPCConfigRequestAction, stack name:", action.stackName)
	cloudformationSrv := cloudformation.New(action.context.Session)

	describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(action.stackName),
	}

	stackResources, err := cloudformationSrv.DescribeStackResources(describeStackResourcesInput)
	if err != nil {
		return nil, err
	}
	stackResourceMap := make(map[string]cloudformation.StackResource)
	for _, res := range stackResources.StackResources {
		stackResourceMap[*res.LogicalResourceId] = *res
	}

	securityGroupResource, found := stackResourceMap["ControlPlaneSecurityGroup"]
	if !found {
		return nil, errors.New("Unable to find ControlPlaneSecurityGroup resource")
	}
	subnet01resource, found := stackResourceMap["Subnet01"]
	if !found {
		return nil, errors.New("Unable to find Subnet02 resource")
	}
	subnet02resource, found := stackResourceMap["Subnet02"]
	if !found {
		return nil, errors.New("Unable to find Subnet01 resource")
	}
	subnet03resource, found := stackResourceMap["Subnet03"]
	if !found {
		return nil, errors.New("Unable to find Subnet03 resource")
	}
	vpcResource, found := stackResourceMap["VPC"]
	if !found {
		return nil, errors.New("Unable to find VPC resource")
	}

	log.Infof("Stack resources: %v", stackResources)

	action.context.VpcID = vpcResource.PhysicalResourceId
	action.context.SecurityGroupID = securityGroupResource.PhysicalResourceId
	action.context.SubnetIDs = []*string{subnet01resource.PhysicalResourceId, subnet02resource.PhysicalResourceId, subnet03resource.PhysicalResourceId}

	return &eks.VpcConfigRequest{
		SecurityGroupIds: []*string{action.context.SecurityGroupID},
		SubnetIds:        action.context.SubnetIDs,
	}, nil
}

// UndoAction rolls back this GenerateVPCConfigRequestAction
func (action *GenerateVPCConfigRequestAction) UndoAction() (err error) {
	log.Infoln("EXECUTE UNDO GenerateVPCConfigRequestAction, stack name:", action.stackName)
	return nil
}

// --

var _ utils.RevocableAction = (*CreateEksClusterAction)(nil)

// CreateEksClusterAction describes the properties of an EKS cluster creation
type CreateEksClusterAction struct {
	context           *EksClusterCreationContext
	kubernetesVersion string
}

// NewCreateEksClusterAction creates a new CreateEksClusterAction
func NewCreateEksClusterAction(creationContext *EksClusterCreationContext, kubernetesVersion string) *CreateEksClusterAction {
	return &CreateEksClusterAction{
		context:           creationContext,
		kubernetesVersion: kubernetesVersion,
	}
}

// GetName returns the name of this CreateEksClusterAction
func (action *CreateEksClusterAction) GetName() string {
	return "CreateEksClusterAction"
}

// ExecuteAction executes this CreateEksClusterAction
func (action *CreateEksClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	vpcConfigRequest, ok := input.(*eks.VpcConfigRequest)

	if !ok {
		return nil, errors.New("input parameter must be a *VpcConfigRequest")
	}
	log.Infoln("EXECUTE CreateEksClusterAction, cluster name:", action.context.ClusterName)
	eksSvc := eks.New(action.context.Session)

	roleArn := action.context.Role.Arn

	createClusterInput := &eks.CreateClusterInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()), //"1d2129a1-3d38-460a-9756-e5b91fddb951"
		Name:               aws.String(action.context.ClusterName),
		ResourcesVpcConfig: vpcConfigRequest,
		RoleArn:            roleArn, //"arn:aws:iam::012345678910:role/eks-service-role-AWSServiceRoleForAmazonEKS-J7ONKE3BQ4PI"
		Version:            aws.String(action.kubernetesVersion),
	}

	result, err := eksSvc.CreateCluster(createClusterInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.Errorf("CreateCluster error [%s]: %s", aerr.Code(), aerr.Error())
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Errorf("CreateCluster error: %s", err.Error())
		}
		return nil, err
	}

	//wait for ready status
	startTime := time.Now()
	log.Info("Waiting for EKS cluster creation")
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(action.context.ClusterName),
	}
	err = action.waitUntilClusterCreateComplete(describeClusterInput)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()
	log.Infoln("EKS cluster created successfully in", endTime.Sub(startTime).String())

	return result.Cluster, nil
}

func (action *CreateEksClusterAction) waitUntilClusterCreateComplete(input *eks.DescribeClusterInput) error {
	return action.waitUntilClusterCreateCompleteWithContext(aws.BackgroundContext(), input)
}

func (action *CreateEksClusterAction) waitUntilClusterCreateCompleteWithContext(ctx aws.Context, input *eks.DescribeClusterInput, opts ...request.WaiterOption) error {
	eksSvc := eks.New(action.context.Session)

	w := request.Waiter{
		Name:        "WaitUntilClusterCreateComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Cluster.Status",
				Expected: eks.ClusterStatusActive,
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Cluster.Status",
				Expected: eks.ClusterStatusDeleting,
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Cluster.Status",
				Expected: eks.ClusterStatusFailed,
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		Logger: eksSvc.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *eks.DescribeClusterInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := eksSvc.DescribeClusterRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}

// UndoAction rolls back this CreateEksClusterAction
func (action *CreateEksClusterAction) UndoAction() (err error) {
	log.Infoln("EXECUTE UNDO CreateEksClusterAction, cluster name", action.context.ClusterName)
	eksSvc := eks.New(action.context.Session)

	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(action.context.ClusterName),
	}
	_, err = eksSvc.DeleteCluster(deleteClusterInput)
	return err
}

// ---

var _ utils.RevocableAction = (*CreateNodePoolStackAction)(nil)

// CreateNodePoolStackAction describes the properties of a nodePool VPC creation
type CreateNodePoolStackAction struct {
	context          *EksClusterCreationContext
	stackName        string
	scalingMinSize   int
	scalingMaxSize   int
	scalingInitSize  int
	autoScaling      bool
	nodeInstanceType string
	nodeImageId      string
	nodeSpotPrice    string
	//describeStacksTimeInterval time.Duration
	//stackCreationTimeout       time.Duration
}

// NewCreateNodePoolStackAction creates a new CreateNodePoolStackAction
func NewCreateNodePoolStackAction(
	creationContext *EksClusterCreationContext,
	stackName string,
	nodePool *model.AmazonNodePoolsModel) *CreateNodePoolStackAction {
	return &CreateNodePoolStackAction{
		context:          creationContext,
		stackName:        stackName,
		scalingMinSize:   nodePool.NodeMinCount,
		scalingMaxSize:   nodePool.NodeMaxCount,
		scalingInitSize:  nodePool.Count,
		autoScaling:      nodePool.Autoscaling,
		nodeInstanceType: nodePool.NodeInstanceType,
		nodeImageId:      nodePool.NodeImage,
		nodeSpotPrice:    nodePool.NodeSpotPrice,
		//describeStacksTimeInterval: 10 * time.Second,
		//stackCreationTimeout:       3 * time.Minute,
	}
}

// GetName return the name of this action
func (action *CreateNodePoolStackAction) GetName() string {
	return "CreateNodePoolStackAction"
}

// ExecuteAction executes the CreateNodePoolStackAction
func (action *CreateNodePoolStackAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE CreateNodePoolStackAction, stack name:", action.stackName)

	templateBody := `
---
AWSTemplateFormatVersion: '2010-09-09'
Description: 'Amazon EKS - Node Group'

Parameters:

  KeyName:
    Description: The EC2 Key Pair to allow SSH access to the instances
    Type: AWS::EC2::KeyPair::KeyName

  NodeImageId:
    Type: AWS::EC2::Image::Id
    Description: AMI id for the node instances.

  NodeInstanceType:
    Description: EC2 instance type for the node instances
    Type: String
    Default: t2.medium
    AllowedValues:
    - t2.small
    - t2.medium
    - t2.large
    - t2.xlarge
    - t2.2xlarge
    - m3.medium
    - m3.large
    - m3.xlarge
    - m3.2xlarge
    - m4.large
    - m4.xlarge
    - m4.2xlarge
    - m4.4xlarge
    - m4.10xlarge
    - m5.large
    - m5.xlarge
    - m5.2xlarge
    - m5.4xlarge
    - m5.12xlarge
    - m5.24xlarge
    - c4.large
    - c4.xlarge
    - c4.2xlarge
    - c4.4xlarge
    - c4.8xlarge
    - c5.large
    - c5.xlarge
    - c5.2xlarge
    - c5.4xlarge
    - c5.9xlarge
    - c5.18xlarge
    - i3.large
    - i3.xlarge
    - i3.2xlarge
    - i3.4xlarge
    - i3.8xlarge
    - i3.16xlarge
    - r3.xlarge
    - r3.2xlarge
    - r3.4xlarge
    - r3.8xlarge
    - r4.large
    - r4.xlarge
    - r4.2xlarge
    - r4.4xlarge
    - r4.8xlarge
    - r4.16xlarge
    - x1.16xlarge
    - x1.32xlarge
    - p2.xlarge
    - p2.8xlarge
    - p2.16xlarge
    - p3.2xlarge
    - p3.8xlarge
    - p3.16xlarge
    ConstraintDescription: must be a valid EC2 instance type

  NodeAutoScalingGroupMinSize:
    Type: Number
    Description: Minimum size of Node Group ASG.
    Default: 1

  NodeAutoScalingGroupMaxSize:
    Type: Number
    Description: Maximum size of Node Group ASG.
    Default: 3

  NodeAutoScalingInitSize:
    Type: Number
    Description: The initial size of Node Group ASG.
    Default: 1

  NodeSpotPrice:
    Type: String
    Description: The spot price for this ASG

  ClusterName:
    Description: The cluster name provided when the cluster was created.  If it is incorrect, nodes will not be able to join the cluster.
    Type: String

  NodeGroupName:
    Description: Unique identifier for the Node Group.
    Type: String

  ClusterControlPlaneSecurityGroup:
    Description: The security group of the cluster control plane.
    Type: AWS::EC2::SecurityGroup::Id

  VpcId:
    Description: The VPC of the worker instances
    Type: AWS::EC2::VPC::Id

  Subnets:
    Description: The subnets where workers can be created.
    Type: List<AWS::EC2::Subnet::Id>

Mappings:
  MaxPodsPerNode:
    c4.large:
      MaxPods: 29
    c4.xlarge:
      MaxPods: 58
    c4.2xlarge:
      MaxPods: 58
    c4.4xlarge:
      MaxPods: 234
    c4.8xlarge:
      MaxPods: 234
    c5.large:
      MaxPods: 29
    c5.xlarge:
      MaxPods: 58
    c5.2xlarge:
      MaxPods: 58
    c5.4xlarge:
      MaxPods: 234
    c5.9xlarge:
      MaxPods: 234
    c5.18xlarge:
      MaxPods: 737
    i3.large:
      MaxPods: 29
    i3.xlarge:
      MaxPods: 58
    i3.2xlarge:
      MaxPods: 58
    i3.4xlarge:
      MaxPods: 234
    i3.8xlarge:
      MaxPods: 234
    i3.16xlarge:
      MaxPods: 737
    m3.medium:
      MaxPods: 12
    m3.large:
      MaxPods: 29
    m3.xlarge:
      MaxPods: 58
    m3.2xlarge:
      MaxPods: 118
    m4.large:
      MaxPods: 20
    m4.xlarge:
      MaxPods: 58
    m4.2xlarge:
      MaxPods: 58
    m4.4xlarge:
      MaxPods: 234
    m4.10xlarge:
      MaxPods: 234
    m5.large:
      MaxPods: 29
    m5.xlarge:
      MaxPods: 58
    m5.2xlarge:
      MaxPods: 58
    m5.4xlarge:
      MaxPods: 234
    m5.12xlarge:
      MaxPods: 234
    m5.24xlarge:
      MaxPods: 737
    p2.xlarge:
      MaxPods: 58
    p2.8xlarge:
      MaxPods: 234
    p2.16xlarge:
      MaxPods: 234
    p3.2xlarge:
      MaxPods: 58
    p3.8xlarge:
      MaxPods: 234
    p3.16xlarge:
      MaxPods: 234
    r3.xlarge:
      MaxPods: 58
    r3.2xlarge:
      MaxPods: 58
    r3.4xlarge:
      MaxPods: 234
    r3.8xlarge:
      MaxPods: 234
    r4.large:
      MaxPods: 29
    r4.xlarge:
      MaxPods: 58
    r4.2xlarge:
      MaxPods: 58
    r4.4xlarge:
      MaxPods: 234
    r4.8xlarge:
      MaxPods: 234
    r4.16xlarge:
      MaxPods: 737
    t2.small:
      MaxPods: 8
    t2.medium:
      MaxPods: 17
    t2.large:
      MaxPods: 35
    t2.xlarge:
      MaxPods: 44
    t2.2xlarge:
      MaxPods: 44
    x1.16xlarge:
      MaxPods: 234
    x1.32xlarge:
      MaxPods: 234

Metadata:
  AWS::CloudFormation::Interface:
    ParameterGroups:
      -
        Label:
          default: "EKS Cluster"
        Parameters:
          - ClusterName
          - ClusterControlPlaneSecurityGroup
      -
        Label:
          default: "Worker Node Configuration"
        Parameters:
          - NodeGroupName
          - NodeAutoScalingGroupMinSize
          - NodeAutoScalingGroupMaxSize
          - NodeAutoScalingInitSize
          - NodeSpotPrice
          - NodeInstanceType
          - NodeImageId
          - KeyName
      -
        Label:
          default: "Worker Network Configuration"
        Parameters:
          - VpcId
          - Subnets
Conditions:
  IsSpotInstance: !Not [ !Equals [ !Ref NodeSpotPrice, "" ] ]

Resources:
  NodeInstanceProfile:
    Type: AWS::IAM::InstanceProfile
    Properties:
      Path: "/"
      Roles:
      - !Ref NodeInstanceRole

  NodeInstanceRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
        - Effect: Allow
          Principal:
            Service:
            - ec2.amazonaws.com
          Action:
          - sts:AssumeRole
      Path: "/"
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy
        - arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy
        - arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly
      Policies:
        -
          PolicyName: NodePolicy
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
            -
              Effect: "Allow"
              Action:
              - ec2:Describe*
              - ecr:GetAuthorizationToken
              - ecr:BatchCheckLayerAvailability
              - ecr:GetDownloadUrlForLayer
              - ecr:GetRepositoryPolicy
              - ecr:DescribeRepositories
              - ecr:ListImages
              - ecr:BatchGetImage
              - s3:ListBucket
              - s3:GetObject
              - s3:PutObject
              - s3:ListObjects
              - s3:DeleteObject
              - autoscaling:DescribeAutoScalingGroups
              - autoscaling:UpdateAutoScalingGroup
              - autoscaling:DescribeAutoScalingInstances
              - autoscaling:DescribeTags
              - autoscaling:DescribeLaunchConfigurations
              - autoscaling:SetDesiredCapacity
              - autoscaling:TerminateInstanceInAutoScalingGroup
              Resource: "*"

  NodeSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Security group for all nodes in the cluster
      VpcId:
        !Ref VpcId
      Tags:
      - Key: !Sub "kubernetes.io/cluster/${ClusterName}"
        Value: 'owned'

  NodeSecurityGroupIngress:
    Type: AWS::EC2::SecurityGroupIngress
    DependsOn: NodeSecurityGroup
    Properties:
      Description: Allow node to communicate with each other
      GroupId: !Ref NodeSecurityGroup
      SourceSecurityGroupId: !Ref NodeSecurityGroup
      IpProtocol: '-1'
      FromPort: 0
      ToPort: 65535

  NodeSecurityGroupFromControlPlaneIngress:
    Type: AWS::EC2::SecurityGroupIngress
    DependsOn: NodeSecurityGroup
    Properties:
      Description: Allow worker Kubelets and pods to receive communication from the cluster control plane
      GroupId: !Ref NodeSecurityGroup
      SourceSecurityGroupId: !Ref ClusterControlPlaneSecurityGroup
      IpProtocol: tcp
      FromPort: 1025
      ToPort: 65535

  ControlPlaneEgressToNodeSecurityGroup:
    Type: AWS::EC2::SecurityGroupEgress
    DependsOn: NodeSecurityGroup
    Properties:
      Description: Allow the cluster control plane to communicate with worker Kubelet and pods
      GroupId: !Ref ClusterControlPlaneSecurityGroup
      DestinationSecurityGroupId: !Ref NodeSecurityGroup
      IpProtocol: tcp
      FromPort: 1025
      ToPort: 65535

  ClusterControlPlaneSecurityGroupIngress:
    Type: AWS::EC2::SecurityGroupIngress
    DependsOn: NodeSecurityGroup
    Properties:
      Description: Allow pods to communicate with the cluster API Server
      GroupId: !Ref ClusterControlPlaneSecurityGroup
      SourceSecurityGroupId: !Ref NodeSecurityGroup
      IpProtocol: tcp
      ToPort: 443
      FromPort: 443

  NodeSecurityGroupSsh:
    Type: AWS::EC2::SecurityGroupIngress
    DependsOn: NodeSecurityGroup
    Properties:
      Description: Allow SSH access to node
      GroupId: !Ref NodeSecurityGroup
      CidrIp: '0.0.0.0/0'
      IpProtocol: tcp
      ToPort: 22
      FromPort: 22

  NodeGroup:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      DesiredCapacity: !Ref NodeAutoScalingInitSize
      LaunchConfigurationName: !Ref NodeLaunchConfig
      MinSize: !Ref NodeAutoScalingGroupMinSize
      MaxSize: !Ref NodeAutoScalingGroupMaxSize
      VPCZoneIdentifier:
        !Ref Subnets
      Tags:
      - Key: Name
        Value: !Sub "${ClusterName}-${NodeGroupName}-Node"
        PropagateAtLaunch: 'true'
      - Key: !Sub 'kubernetes.io/cluster/${ClusterName}'
        Value: 'owned'
        PropagateAtLaunch: 'true'

    UpdatePolicy:
      AutoScalingRollingUpdate:
        MinInstancesInService: '1'
        MaxBatchSize: '1'

  NodeLaunchConfig:
    Type: AWS::AutoScaling::LaunchConfiguration
    Properties:
      AssociatePublicIpAddress: 'true'
      IamInstanceProfile: !Ref NodeInstanceProfile
      ImageId: !Ref NodeImageId
      InstanceType: !Ref NodeInstanceType
      SpotPrice: !If [ IsSpotInstance, !Ref NodeSpotPrice, !Ref "AWS::NoValue" ]
      KeyName: !Ref KeyName
      SecurityGroups:
      - !Ref NodeSecurityGroup
      UserData:
        Fn::Base64:
          Fn::Join: [
            "",
            [
              "#!/bin/bash -xe\n",
              "CA_CERTIFICATE_DIRECTORY=/etc/kubernetes/pki", "\n",
              "CA_CERTIFICATE_FILE_PATH=$CA_CERTIFICATE_DIRECTORY/ca.crt", "\n",
              "MODEL_DIRECTORY_PATH=~/.aws/eks", "\n",
              "MODEL_FILE_PATH=$MODEL_DIRECTORY_PATH/eks-2017-11-01.normal.json", "\n",
              "mkdir -p $CA_CERTIFICATE_DIRECTORY", "\n",
              "mkdir -p $MODEL_DIRECTORY_PATH", "\n",
              "curl -o $MODEL_FILE_PATH https://s3-us-west-2.amazonaws.com/amazon-eks/1.10.3/2018-06-05/eks-2017-11-01.normal.json", "\n",
              "aws configure add-model --service-model file://$MODEL_FILE_PATH --service-name eks", "\n",
              "aws eks describe-cluster --region=", { Ref: "AWS::Region" }," --name=", { Ref: ClusterName }," --query 'cluster.{certificateAuthorityData: certificateAuthority.data, endpoint: endpoint}' > /tmp/describe_cluster_result.json", "\n",
              "cat /tmp/describe_cluster_result.json | grep certificateAuthorityData | awk '{print $2}' | sed 's/[,\"]//g' | base64 -d >  $CA_CERTIFICATE_FILE_PATH", "\n",
              "MASTER_ENDPOINT=$(cat /tmp/describe_cluster_result.json | grep endpoint | awk '{print $2}' | sed 's/[,\"]//g')", "\n",
              "INTERNAL_IP=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4)", "\n",
              "sed -i s,MASTER_ENDPOINT,$MASTER_ENDPOINT,g /var/lib/kubelet/kubeconfig", "\n",
              "sed -i s,CLUSTER_NAME,", { Ref: ClusterName }, ",g /var/lib/kubelet/kubeconfig", "\n",
              "sed -i s,REGION,", { Ref: "AWS::Region" }, ",g /etc/systemd/system/kubelet.service", "\n",
              "sed -i s,MAX_PODS,", { "Fn::FindInMap": [ MaxPodsPerNode, { Ref: NodeInstanceType }, MaxPods ] }, ",g /etc/systemd/system/kubelet.service", "\n",
              "sed -i s,MASTER_ENDPOINT,$MASTER_ENDPOINT,g /etc/systemd/system/kubelet.service", "\n",
              "sed -i s,INTERNAL_IP,$INTERNAL_IP,g /etc/systemd/system/kubelet.service", "\n",
              "DNS_CLUSTER_IP=10.100.0.10", "\n",
              "if [[ $INTERNAL_IP == 10.* ]] ; then DNS_CLUSTER_IP=172.20.0.10; fi", "\n",
              "sed -i s,DNS_CLUSTER_IP,$DNS_CLUSTER_IP,g  /etc/systemd/system/kubelet.service", "\n",
              "sed -i s,CERTIFICATE_AUTHORITY_FILE,$CA_CERTIFICATE_FILE_PATH,g /var/lib/kubelet/kubeconfig" , "\n",
              "sed -i s,CLIENT_CA_FILE,$CA_CERTIFICATE_FILE_PATH,g  /etc/systemd/system/kubelet.service" , "\n",
              "systemctl daemon-reload", "\n",
              "systemctl restart kubelet", "\n",
              "/opt/aws/bin/cfn-signal -e $? ",
              "         --stack ", { Ref: "AWS::StackName" },
              "         --resource NodeGroup ",
              "         --region ", { Ref: "AWS::Region" }, "\n"
            ]
          ]

Outputs:
  NodeInstanceRole:
    Description: The node instance role
    Value: !GetAtt NodeInstanceRole.Arn
`
	commaDelimitedSubnetIDs := ""
	for i, subnetID := range action.context.SubnetIDs {
		commaDelimitedSubnetIDs = commaDelimitedSubnetIDs + *subnetID
		if i != len(action.context.SubnetIDs)-1 {
			commaDelimitedSubnetIDs = commaDelimitedSubnetIDs + ","
		}
	}

	tags := []*cloudformation.Tag{
		{Key: aws.String("pipeline-created"), Value: aws.String("true")},
	}

	if action.autoScaling {
		tags = append(tags, &cloudformation.Tag{Key: aws.String("k8s.io/cluster-autoscaler/enabled"), Value: aws.String("true")})
	}

	spotPriceParam := ""
	if p, err := strconv.ParseFloat(action.nodeSpotPrice, 64); err == nil && p > 0.0 {
		spotPriceParam = action.nodeSpotPrice
	}

	cloudformationSrv := cloudformation.New(action.context.Session)
	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		DisableRollback:    aws.Bool(false),
		StackName:          aws.String(action.stackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String(action.context.SSHKeyName),
			},
			{
				ParameterKey:   aws.String("NodeImageId"),
				ParameterValue: aws.String(action.nodeImageId),
			},
			{
				ParameterKey:   aws.String("NodeInstanceType"),
				ParameterValue: aws.String(action.nodeInstanceType),
			},
			{
				ParameterKey:   aws.String("NodeSpotPrice"),
				ParameterValue: aws.String(spotPriceParam),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", action.scalingMinSize)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", action.scalingMaxSize)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingInitSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", action.scalingInitSize)),
			},
			{
				ParameterKey:   aws.String("ClusterName"),
				ParameterValue: aws.String(action.context.ClusterName),
			},
			{
				ParameterKey:   aws.String("NodeGroupName"),
				ParameterValue: aws.String(fmt.Sprintf("%s%s", action.stackName, "-nodegroup")),
			},
			{
				ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
				ParameterValue: action.context.SecurityGroupID,
			},
			{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: action.context.VpcID,
			}, {
				ParameterKey:   aws.String("Subnets"),
				ParameterValue: aws.String(commaDelimitedSubnetIDs),
			},
		},
		Tags:             tags,
		TemplateBody:     aws.String(templateBody),
		TimeoutInMinutes: aws.Int64(10),
	}

	//startTime := time.Now()
	_, err = cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		return
	}

	//for time.Now().Before(startTime.Add(action.stackCreationTimeout)) {
	//	stackStatuses, err := cloudformationSrv.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(action.stackName)})
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	if len(stackStatuses.Stacks) != 1 {
	//		return nil, errors.New(fmt.Sprintf("Got %d stack(s) instead of 1 after stack creation", len(stackStatuses.Stacks)))
	//	}
	//	stack := stackStatuses.Stacks[0]
	//	fmt.Printf("stackStatus: %s\n", *stack.StackStatus)
	//	if *stack.StackStatus == cloudformation.StackStatusCreateComplete {
	//		completed = true
	//		break
	//	}
	//	time.Sleep(action.describeStacksTimeInterval)
	//}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(action.stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)

	if err != nil {
		return nil, err
	}

	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		return
	}

	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		if aws.StringValue(output.OutputKey) == "NodeInstanceRole" {
			action.context.NodeInstanceRoles = append(action.context.NodeInstanceRoles, aws.StringValue(output.OutputValue))
		}
	}

	return nil, nil
}

// UndoAction rolls back this CreateNodePoolStackAction
func (action *CreateNodePoolStackAction) UndoAction() (err error) {
	log.Info("EXECUTE UNDO CreateNodePoolStackAction")
	cloudformationSrv := cloudformation.New(action.context.Session)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		StackName:          aws.String(action.stackName),
	}
	_, err = cloudformationSrv.DeleteStack(deleteStackInput)

	//TODO delete each created object
	return
}

// ---

var _ utils.RevocableAction = (*UploadSSHKeyAction)(nil)

// UploadSSHKeyAction describes how to upload an SSH key
type UploadSSHKeyAction struct {
	context   *EksClusterCreationContext
	sshSecret *secret.SecretItemResponse
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(context *EksClusterCreationContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
	return &UploadSSHKeyAction{
		context:   context,
		sshSecret: sshSecret,
	}
}

// GetName returns the name of this UploadSSHKeyAction
func (action *UploadSSHKeyAction) GetName() string {
	return "UploadSSHKeyAction"
}

// ExecuteAction executes this UploadSSHKeyAction
func (action *UploadSSHKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE UploadSSHKeyAction")

	action.context.SSHKey = secret.NewSSHKeyPair(action.sshSecret)
	ec2srv := ec2.New(action.context.Session)
	importKeyPairInput := &ec2.ImportKeyPairInput{

		// A unique name for the key pair.
		// KeyName is a required field
		KeyName: aws.String(action.context.SSHKeyName),

		// The public key. For API calls, the text must be base64-encoded. For command
		// line tools, base64 encoding is performed for you.
		//
		// PublicKeyMaterial is automatically base64 encoded/decoded by the SDK.
		//
		// PublicKeyMaterial is a required field
		PublicKeyMaterial: []byte(action.context.SSHKey.PublicKeyData), // []byte `locationName:"publicKeyMaterial" type:"blob" required:"true"`
	}
	output, err = ec2srv.ImportKeyPair(importKeyPairInput)
	return output, err
}

// UndoAction rolls back this UploadSSHKeyAction
func (action *UploadSSHKeyAction) UndoAction() (err error) {
	log.Info("EXECUTE UNDO UploadSSHKeyAction")
	//delete uploaded keypair
	ec2srv := ec2.New(action.context.Session)

	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(action.context.SSHKeyName),
	}
	_, err = ec2srv.DeleteKeyPair(deleteKeyPairInput)
	return err
}

// ---

var _ utils.RevocableAction = (*RevertStepsAction)(nil)

// RevertStepsAction can be used to intentionally revert all the steps (=simulate an error)
type RevertStepsAction struct {
}

// NewRevertStepsAction creates a new RevertStepsAction
func NewRevertStepsAction() *RevertStepsAction {
	return &RevertStepsAction{}
}

// GetName returns the name of this RevertStepsAction
func (action *RevertStepsAction) GetName() string {
	return "RevertStepsAction"
}

// ExecuteAction executes this RevertStepsAction
func (action *RevertStepsAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE RevertStepsAction")
	return nil, errors.New("Intentionally reverting everything")
}

// UndoAction rolls back this RevertStepsAction
func (action *RevertStepsAction) UndoAction() (err error) {
	log.Info("EXECUTE UNDO RevertStepsAction")
	return nil
}

// ---

var _ utils.RevocableAction = (*DelayAction)(nil)

// DelayAction can be used to intentionally delay the next step
type DelayAction struct {
	delay time.Duration
}

// NewDelayAction creates a new DelayAction
func NewDelayAction(delay time.Duration) *DelayAction {
	return &DelayAction{
		delay: delay,
	}
}

// GetName returns the name of this DelayAction
func (action *DelayAction) GetName() string {
	return "DelayAction"
}

// ExecuteAction executes this DelayAction
func (action *DelayAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE DelayAction")
	time.Sleep(action.delay)
	return input, nil
}

// UndoAction rolls back this DelayAction
func (action *DelayAction) UndoAction() (err error) {
	log.Info("EXECUTE UNDO RevertStepsAction")
	return nil
}

// ---

var _ utils.RevocableAction = (*LoadEksSettingsAction)(nil)

// LoadEksSettingsAction to describe the EKS cluster created
type LoadEksSettingsAction struct {
	context *EksClusterCreationContext
}

// NewLoadEksSettingsAction creates a new LoadEksSettingsAction
func NewLoadEksSettingsAction(context *EksClusterCreationContext) *LoadEksSettingsAction {
	return &LoadEksSettingsAction{
		context: context,
	}
}

// GetName returns the name of this LoadEksSettingsAction
func (action *LoadEksSettingsAction) GetName() string {
	return "LoadEksSettingsAction"
}

// ExecuteAction executes this LoadEksSettingsAction
func (action *LoadEksSettingsAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE LoadEksSettingsAction")
	eksSvc := eks.New(action.context.Session)
	//Store API endpoint, etc..
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(action.context.ClusterName),
	}
	clusterInfo, err := eksSvc.DescribeCluster(describeClusterInput) //...ha kellene vmi info az eks clusterrol
	if err != nil {
		return nil, err
	}
	cluster := clusterInfo.Cluster
	if cluster == nil {
		return nil, errors.New("Unable to get EKS Cluster info")
	}

	action.context.APIEndpoint = cluster.Endpoint
	action.context.CertificateAuthorityData = cluster.CertificateAuthority.Data
	//TODO store settings in db

	return input, nil
}

// UndoAction rolls back this LoadEksSettingsAction
func (action *LoadEksSettingsAction) UndoAction() (err error) {
	log.Info("EXECUTE UNDO LoadEksSettingsAction")
	return nil
}

//--

var _ utils.Action = (*DeleteStackAction)(nil)

// DeleteStackAction deletes a stack
type DeleteStackAction struct {
	context   *EksClusterDeletionContext
	StackName string
}

// NewDeleteStackAction creates a new DeleteStackAction
func NewDeleteStackAction(context *EksClusterDeletionContext, stackName string) *DeleteStackAction {
	return &DeleteStackAction{
		context:   context,
		StackName: stackName,
	}
}

// GetName returns the name of this DeleteStackAction
func (action *DeleteStackAction) GetName() string {
	return "DeleteStackAction"
}

// ExecuteAction executes this DeleteStackAction
func (action *DeleteStackAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE DeleteStackAction")

	//TODO handle non existing stack
	cloudformationSrv := cloudformation.New(action.context.Session)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		StackName:          aws.String(action.StackName),
	}
	return cloudformationSrv.DeleteStack(deleteStackInput)
}

//--

var _ utils.Action = (*DeleteClusterAction)(nil)

// DeleteClusterAction deletes an EKS cluster
type DeleteClusterAction struct {
	context        *EksClusterDeletionContext
	EksClusterName string
}

// NewDeleteClusterAction creates a new DeleteClusterAction
func NewDeleteClusterAction(context *EksClusterDeletionContext, eksClusterName string) *DeleteClusterAction {
	return &DeleteClusterAction{
		context:        context,
		EksClusterName: eksClusterName,
	}
}

// GetName returns the name of this DeleteClusterAction
func (action *DeleteClusterAction) GetName() string {
	return "DeleteClusterAction"
}

// ExecuteAction executes this DeleteClusterAction
func (action *DeleteClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE DeleteClusterAction")

	//TODO handle non existing cluster
	eksSrv := eks.New(action.context.Session)
	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(action.EksClusterName),
	}
	return eksSrv.DeleteCluster(deleteClusterInput)
}

//--

var _ utils.Action = (*DeleteSSHKeyAction)(nil)

// DeleteSSHKeyAction deletes a generated SSH key
type DeleteSSHKeyAction struct {
	context    *EksClusterDeletionContext
	SSHKeyName string
}

// NewDeleteSSHKeyAction creates a new DeleteSSHKeyAction
func NewDeleteSSHKeyAction(context *EksClusterDeletionContext, sshKeyName string) *DeleteSSHKeyAction {
	return &DeleteSSHKeyAction{
		context:    context,
		SSHKeyName: sshKeyName,
	}
}

// GetName returns the name of this DeleteSSHKeyAction
func (action *DeleteSSHKeyAction) GetName() string {
	return "DeleteSSHKeyAction"
}

// ExecuteAction executes this DeleteSSHKeyAction
func (action *DeleteSSHKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE DeleteSSHKeyAction")

	//TODO handle non existing key
	ec2srv := ec2.New(action.context.Session)
	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(action.SSHKeyName),
	}
	output, err = ec2srv.DeleteKeyPair(deleteKeyPairInput)
	return output, err
}

//--

var _ utils.Action = (*DeleteIAMRoleAction)(nil)

// DeleteIAMRoleAction deletes an IAM role
type DeleteIAMRoleAction struct {
	context  *EksClusterDeletionContext
	RoleName string
}

// NewDeleteIAMRoleAction creates a new DeleteIAMRoleAction
func NewDeleteIAMRoleAction(context *EksClusterDeletionContext, roleName string) *DeleteIAMRoleAction {
	return &DeleteIAMRoleAction{
		context:  context,
		RoleName: roleName,
	}
}

// GetName returns the name of this DeleteIAMRoleAction
func (action *DeleteIAMRoleAction) GetName() string {
	return "DeleteIAMRoleAction"
}

// ExecuteAction executes this DeleteIAMRoleAction
func (action *DeleteIAMRoleAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE DeleteIAMRoleAction, deleting role:", action.RoleName)

	//TODO handle non existing role
	// detach every role first
	// then delete role

	iamSvc := iam.New(action.context.Session)

	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(action.RoleName),
	}

	_, err = iamSvc.GetRole(getRoleInput)
	if err != nil {
		return nil, err
	}

	// For managed policies
	managedPolicies := make([]*string, 0)
	err = iamSvc.ListAttachedRolePoliciesPages(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(action.RoleName),
	}, func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
		for _, v := range page.AttachedPolicies {
			managedPolicies = append(managedPolicies, v.PolicyArn)
		}
		return len(page.AttachedPolicies) > 0
	})

	if err != nil {
		log.Debug("ListAttachedRolePoliciesPages error: %v", err)
		return nil, err
	}

	//detach role policies first
	for _, policyName := range managedPolicies {
		detachRolePolicyInput := &iam.DetachRolePolicyInput{
			RoleName:  aws.String(action.RoleName),
			PolicyArn: policyName, //TODO should we use ARN here?
		}
		_, err = iamSvc.DetachRolePolicy(detachRolePolicyInput)
		if err != nil {
			log.Debug("DetachRolePolicy error: %v", err)
			return nil, err
		}
	}
	//delete role
	deleteRoleInput := &iam.DeleteRoleInput{
		RoleName: aws.String(action.RoleName),
	}
	_, err = iamSvc.DeleteRole(deleteRoleInput)
	return nil, err
}

//--

var _ utils.Action = (*DeleteUserAction)(nil)

// DeleteUserAction deletes an IAM role
type DeleteUserAction struct {
	context     *EksClusterDeletionContext
	userName    string
	accessKeyID string
}

// NewDeleteUserAction creates a new DeleteUserAction
func NewDeleteUserAction(context *EksClusterDeletionContext, userName, accessKeyID string) *DeleteUserAction {
	return &DeleteUserAction{
		context:     context,
		userName:    userName,
		accessKeyID: accessKeyID,
	}
}

// GetName returns the name of this DeleteUserAction
func (action *DeleteUserAction) GetName() string {
	return "DeleteUserAction"
}

// ExecuteAction executes this DeleteUserAction
func (action *DeleteUserAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE DeleteUserAction, deleting user:", action.userName)

	iamSvc := iam.New(action.context.Session)

	deleteAccessKeyInput := &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(action.accessKeyID),
		UserName:    aws.String(action.userName),
	}

	_, err = iamSvc.DeleteAccessKey(deleteAccessKeyInput)
	if err != nil {
		return nil, err
	}

	deleteUserInput := &iam.DeleteUserInput{
		UserName: aws.String(action.userName),
	}

	_, err = iamSvc.DeleteUser(deleteUserInput)

	return nil, err
}
