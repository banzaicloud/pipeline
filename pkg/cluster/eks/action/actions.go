package action

import (
	"fmt"
	"time"

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
	NodeInstanceRole         string
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
		Tags:             []*cloudformation.Tag{&cloudformation.Tag{Key: aws.String("pipeline-created"), Value: aws.String("true")}},
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
	nodeInstanceType string
	nodeImageId      string
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
		nodeInstanceType: nodePool.NodeInstanceType,
		nodeImageId:      nodePool.NodeImage,
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

	commaDelimitedSubnetIDs := ""
	for i, subnetID := range action.context.SubnetIDs {
		commaDelimitedSubnetIDs = commaDelimitedSubnetIDs + *subnetID
		if i != len(action.context.SubnetIDs)-1 {
			commaDelimitedSubnetIDs = commaDelimitedSubnetIDs + ","
		}
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
				ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", action.scalingMinSize)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", action.scalingMaxSize)),
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
		Tags:             []*cloudformation.Tag{{Key: aws.String("pipeline-created"), Value: aws.String("true")}},
		TemplateURL:      aws.String("https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-06-05/amazon-eks-nodegroup.yaml"),
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
		if *output.OutputKey == "NodeInstanceRole" {
			action.context.NodeInstanceRole = *output.OutputValue
		}
	}

	if action.context.NodeInstanceRole == "" {
		return nil, fmt.Errorf("Failed to find NodeInstanceRole")
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
