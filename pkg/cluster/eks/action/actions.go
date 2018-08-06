package action

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

const awsNoUpdatesError = "No updates are to be performed."

// Simple init for logging
func init() {
	log = config.Logger()
}

// --

// EksClusterCreateUpdateContext describes the properties of an EKS cluster creation
type EksClusterCreateUpdateContext struct {
	sync.Mutex
	Session                  *session.Session
	ClusterName              string
	ClusterRoleArn           string
	NodeInstanceRoleID       *string
	NodeInstanceRoleArn      string
	SecurityGroupID          *string
	SubnetIDs                []*string
	SSHKeyName               string
	SSHKey                   *secret.SSHKeyPair
	VpcID                    *string
	ProvidedRoleArn          string
	APIEndpoint              *string
	CertificateAuthorityData *string
}

// NewEksClusterCreationContext creates a new EksClusterCreateUpdateContext
func NewEksClusterCreationContext(session *session.Session, clusterName string, sshKeyName string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		Session:     session,
		ClusterName: clusterName,
		SSHKeyName:  sshKeyName,
	}
}

// NewEksClusterUpdateContext creates a new EksClusterCreateUpdateContext
func NewEksClusterUpdateContext(session *session.Session, clusterName string,
	securityGroupID *string, subnetIDs []*string, sshKeyName string, vpcID *string, nodeInstanceRoleId *string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		Session:            session,
		ClusterName:        clusterName,
		SecurityGroupID:    securityGroupID,
		SubnetIDs:          subnetIDs,
		SSHKeyName:         sshKeyName,
		VpcID:              vpcID,
		NodeInstanceRoleID: nodeInstanceRoleId,
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

var _ utils.RevocableAction = (*CreateVPCAndRolesAction)(nil)

// CreateVPCAndRolesAction describes the properties of a VPC creation
type CreateVPCAndRolesAction struct {
	context   *EksClusterCreateUpdateContext
	stackName string
	//describeStacksTimeInterval time.Duration
	//stackCreationTimeout       time.Duration
}

// NewCreateVPCAndRolesAction creates a new CreateVPCAndRolesAction
func NewCreateVPCAndRolesAction(creationContext *EksClusterCreateUpdateContext, stackName string) *CreateVPCAndRolesAction {
	return &CreateVPCAndRolesAction{
		context:   creationContext,
		stackName: stackName,
		//describeStacksTimeInterval: 10 * time.Second,
		//stackCreationTimeout:       3 * time.Minute,
	}
}

// GetName returns the name of this CreateVPCAndRolesAction
func (action *CreateVPCAndRolesAction) GetName() string {
	return "CreateVPCAndRolesAction"
}

// ExecuteAction executes this CreateVPCAndRolesAction
func (action *CreateVPCAndRolesAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Infoln("EXECUTE CreateVPCAndRolesAction, stack name:", action.stackName)

	log.Infoln("Getting CloudFormation template for creating VPC for EKS cluster")
	templateBody, err := pkgEks.GetVPCTemplate()
	if err != nil {
		log.Errorln("Getting CloudFormation template for VPC failed:", err.Error())
		return nil, err
	}

	cloudformationSrv := cloudformation.New(action.context.Session)
	createStackInput := &cloudformation.CreateStackInput{
		//Capabilities:       []*string{},
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		DisableRollback:    aws.Bool(false),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		StackName:          aws.String(action.stackName),
		Tags:               []*cloudformation.Tag{{Key: aws.String("pipeline-created"), Value: aws.String("true")}},
		TemplateBody:       aws.String(templateBody),
		TimeoutInMinutes:   aws.Int64(10),
	}
	_, err = cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		return
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(action.stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)

	return nil, err
}

// UndoAction rolls back this CreateVPCAndRolesAction
func (action *CreateVPCAndRolesAction) UndoAction() (err error) {
	log.Infoln("EXECUTE UNDO CreateVPCAndRolesAction, deleting stack:", action.stackName)
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
	context   *EksClusterCreateUpdateContext
	stackName string
}

// NewGenerateVPCConfigRequestAction creates a new GenerateVPCConfigRequestAction
func NewGenerateVPCConfigRequestAction(creationContext *EksClusterCreateUpdateContext, stackName string) *GenerateVPCConfigRequestAction {
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
	nodeInstanceProfileResource, found := stackResourceMap["NodeInstanceRole"]
	if !found {
		return nil, errors.New("Unable to find NodeInstanceRole resource")
	}

	log.Infof("Stack resources: %v", stackResources)

	action.context.VpcID = vpcResource.PhysicalResourceId
	action.context.SecurityGroupID = securityGroupResource.PhysicalResourceId
	action.context.SubnetIDs = []*string{subnet01resource.PhysicalResourceId, subnet02resource.PhysicalResourceId, subnet03resource.PhysicalResourceId}
	action.context.NodeInstanceRoleID = nodeInstanceProfileResource.PhysicalResourceId

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(action.stackName)}
	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.New("Unable to find stack " + action.stackName)
	}

	var clusterRoleArn, nodeInstanceRoleArn string
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch *output.OutputKey {
		case "ClusterRoleArn":
			clusterRoleArn = *output.OutputValue
		case "NodeInstanceRoleArn":
			nodeInstanceRoleArn = *output.OutputValue
		}
	}
	log.Infof("cluster role ARN: %v", clusterRoleArn)
	action.context.ClusterRoleArn = clusterRoleArn

	log.Infof("nodeInstanceRoleArn role ARN: %v", nodeInstanceRoleArn)
	action.context.NodeInstanceRoleArn = nodeInstanceRoleArn

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
	context           *EksClusterCreateUpdateContext
	kubernetesVersion string
}

// NewCreateEksClusterAction creates a new CreateEksClusterAction
func NewCreateEksClusterAction(creationContext *EksClusterCreateUpdateContext, kubernetesVersion string) *CreateEksClusterAction {
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

	roleArn := action.context.ClusterRoleArn

	createClusterInput := &eks.CreateClusterInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		Name:               aws.String(action.context.ClusterName),
		ResourcesVpcConfig: vpcConfigRequest,
		RoleArn:            &roleArn,
	}

	// set Kubernetes version only if provided, otherwise the cloud provider default one will be used
	if len(action.kubernetesVersion) > 0 {
		createClusterInput.Version = aws.String(action.kubernetesVersion)
	}

	result, err := eksSvc.CreateCluster(createClusterInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.Errorf("CreateCluster error [%s]: %s", aerr.Code(), aerr.Error())
		} else {
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

var _ utils.RevocableAction = (*CreateUpdateNodePoolStackAction)(nil)

// CreateUpdateNodePoolStackAction describes the properties of a nodePool VPC creation
type CreateUpdateNodePoolStackAction struct {
	context   *EksClusterCreateUpdateContext
	isCreate  bool
	nodePools []*model.AmazonNodePoolsModel
}

// NewCreateUpdateNodePoolStackAction creates a new CreateUpdateNodePoolStackAction
func NewCreateUpdateNodePoolStackAction(
	isCreate bool,
	creationContext *EksClusterCreateUpdateContext,
	nodePools ...*model.AmazonNodePoolsModel) *CreateUpdateNodePoolStackAction {
	return &CreateUpdateNodePoolStackAction{
		context:   creationContext,
		isCreate:  isCreate,
		nodePools: nodePools,
	}
}

func (action *CreateUpdateNodePoolStackAction) generateStackName(nodePool *model.AmazonNodePoolsModel) string {
	return action.context.ClusterName + "-pipeline-eks-nodepool-" + nodePool.Name
}

// GetName return the name of this action
func (action *CreateUpdateNodePoolStackAction) GetName() string {
	return "CreateUpdateNodePoolStackAction"
}

// ExecuteAction executes the CreateUpdateNodePoolStackAction in parallel for each node pool
func (action *CreateUpdateNodePoolStackAction) ExecuteAction(input interface{}) (output interface{}, err error) {

	errorChan := make(chan error, len(action.nodePools))

	for _, nodePool := range action.nodePools {

		go func(nodePool *model.AmazonNodePoolsModel) {

			stackName := action.generateStackName(nodePool)

			if action.isCreate {
				log.Infoln("EXECUTE CreateUpdateNodePoolStackAction, create stack name:", stackName)
			} else {
				log.Infoln("EXECUTE CreateUpdateNodePoolStackAction, update stack name:", stackName)
			}

			templateBody := ""
			if action.isCreate {
				log.Infoln("Getting CloudFormation template for creating node pools for EKS cluster")
				templateBody, err = pkgEks.GetNodePoolTemplate()
				if err != nil {
					log.Errorln("Getting CloudFormation template for node pools failed: ", err.Error())
					errorChan <- err
					return
				}
			}

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

			if nodePool.Autoscaling {
				tags = append(tags, &cloudformation.Tag{Key: aws.String("k8s.io/cluster-autoscaler/enabled"), Value: aws.String("true")})
			}

			spotPriceParam := ""
			if p, err := strconv.ParseFloat(nodePool.NodeSpotPrice, 64); err == nil && p > 0.0 {
				spotPriceParam = nodePool.NodeSpotPrice
			}

			stackParams := []*cloudformation.Parameter{
				{
					ParameterKey:   aws.String("KeyName"),
					ParameterValue: aws.String(action.context.SSHKeyName),
				},
				{
					ParameterKey:   aws.String("NodeImageId"),
					ParameterValue: aws.String(nodePool.NodeImage),
				},
				{
					ParameterKey:   aws.String("NodeInstanceType"),
					ParameterValue: aws.String(nodePool.NodeInstanceType),
				},
				{
					ParameterKey:   aws.String("NodeSpotPrice"),
					ParameterValue: aws.String(spotPriceParam),
				},
				{
					ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
					ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.NodeMinCount)),
				},
				{
					ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
					ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.NodeMaxCount)),
				},
				{
					ParameterKey:   aws.String("NodeAutoScalingInitSize"),
					ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.Count)),
				},
				{
					ParameterKey:   aws.String("ClusterName"),
					ParameterValue: aws.String(action.context.ClusterName),
				},
				{
					ParameterKey:   aws.String("NodeGroupName"),
					ParameterValue: aws.String(nodePool.Name),
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
				{
					ParameterKey:   aws.String("NodeInstanceRoleId"),
					ParameterValue: action.context.NodeInstanceRoleID,
				},
			}

			cloudformationSrv := cloudformation.New(action.context.Session)

			waitOnCreateUpdte := true

			// create stack
			if action.isCreate {
				createStackInput := &cloudformation.CreateStackInput{
					ClientRequestToken: aws.String(uuid.NewV4().String()),
					DisableRollback:    aws.Bool(false),
					StackName:          aws.String(stackName),
					Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
					Parameters:         stackParams,
					Tags:               tags,
					TemplateBody:       aws.String(templateBody),
					TimeoutInMinutes:   aws.Int64(10),
				}
				_, err = cloudformationSrv.CreateStack(createStackInput)
				if err != nil {
					errorChan <- err
					return
				}
			} else {
				// update stack
				reuseTemplate := true
				updateStackInput := &cloudformation.UpdateStackInput{
					ClientRequestToken:  aws.String(uuid.NewV4().String()),
					StackName:           aws.String(stackName),
					Capabilities:        []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
					Parameters:          stackParams,
					Tags:                tags,
					UsePreviousTemplate: &reuseTemplate,
				}

				_, err = cloudformationSrv.UpdateStack(updateStackInput)
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationError" && strings.HasPrefix(awsErr.Message(), awsNoUpdatesError) {
						// Get error details
						log.Warnf("Nothing changed during update!")
						waitOnCreateUpdte = false
						err = nil
					} else {
						errorChan <- err
						return
					}
				}
			}

			describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}

			if action.isCreate {
				err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)
			} else if waitOnCreateUpdte {
				err = cloudformationSrv.WaitUntilStackUpdateComplete(describeStacksInput)
			}

			if err != nil {
				errorChan <- err
				return
			}

			_, err := cloudformationSrv.DescribeStacks(describeStacksInput)
			if err != nil {
				errorChan <- err
				return
			}

			errorChan <- nil

		}(nodePool)
	}

	// wait for goroutines to finish
	for i := 0; i < len(action.nodePools); i++ {
		createErr := <-errorChan
		if createErr != nil {
			err = createErr
		}
	}

	return nil, err
}

// UndoAction rolls back this CreateUpdateNodePoolStackAction
func (action *CreateUpdateNodePoolStackAction) UndoAction() (err error) {
	for _, nodepool := range action.nodePools {
		log.Info("EXECUTE UNDO CreateUpdateNodePoolStackAction")
		cloudformationSrv := cloudformation.New(action.context.Session)
		deleteStackInput := &cloudformation.DeleteStackInput{
			ClientRequestToken: aws.String(uuid.NewV4().String()),
			StackName:          aws.String(action.generateStackName(nodepool)),
		}
		_, deleteErr := cloudformationSrv.DeleteStack(deleteStackInput)
		if deleteErr != nil {
			log.Errorln("Error during deleting CloudFormation stack:", err.Error())
			err = deleteErr
		}
	}
	//TODO delete each created object
	return
}

// ---

var _ utils.RevocableAction = (*UploadSSHKeyAction)(nil)

// UploadSSHKeyAction describes how to upload an SSH key
type UploadSSHKeyAction struct {
	context   *EksClusterCreateUpdateContext
	sshSecret *secret.SecretItemResponse
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(context *EksClusterCreateUpdateContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
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
	context *EksClusterCreateUpdateContext
}

// NewLoadEksSettingsAction creates a new LoadEksSettingsAction
func NewLoadEksSettingsAction(context *EksClusterCreateUpdateContext) *LoadEksSettingsAction {
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
	context *EksClusterDeletionContext
}

// NewDeleteClusterAction creates a new DeleteClusterAction
func NewDeleteClusterAction(context *EksClusterDeletionContext) *DeleteClusterAction {
	return &DeleteClusterAction{
		context: context,
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
		Name: aws.String(action.context.ClusterName),
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

var _ utils.Action = (*WaitResourceDeletionAction)(nil)

// WaitResourceDeletionAction deletes a generated SSH key
type WaitResourceDeletionAction struct {
	context *EksClusterDeletionContext
}

// NewWaitResourceDeletionAction creates a new WaitResourceDeletionAction
func NewWaitResourceDeletionAction(context *EksClusterDeletionContext) *WaitResourceDeletionAction {
	return &WaitResourceDeletionAction{
		context: context,
	}
}

// GetName returns the name of this WaitResourceDeletionAction
func (action *WaitResourceDeletionAction) GetName() string {
	return "WaitResourceDeletionAction"
}

// ExecuteAction executes this WaitResourceDeletionAction
func (action *WaitResourceDeletionAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	log.Info("EXECUTE WaitResourceDeletionAction")

	return nil, action.waitUntilELBsDeleted()
}

func (action *WaitResourceDeletionAction) waitUntilELBsDeleted() error {

	elbService := elb.New(action.context.Session)
	clusterTag := "kubernetes.io/cluster/" + action.context.ClusterName

	for {

		describeLoadBalancers := &elb.DescribeLoadBalancersInput{}
		loadBalancers, err := elbService.DescribeLoadBalancers(describeLoadBalancers)
		if err != nil {
			return err
		}

		var loadBalancerNames []*string
		for _, description := range loadBalancers.LoadBalancerDescriptions {
			loadBalancerNames = append(loadBalancerNames, description.LoadBalancerName)
		}

		if len(loadBalancerNames) == 0 {
			return nil
		}

		describeTagsInput := &elb.DescribeTagsInput{
			LoadBalancerNames: loadBalancerNames,
		}
		describeTagsOutput, err := elbService.DescribeTags(describeTagsInput)
		if err != nil {
			return err
		}

		var result []*string
		for _, tagDescription := range describeTagsOutput.TagDescriptions {
			for _, tag := range tagDescription.Tags {
				if aws.StringValue(tag.Key) == clusterTag {
					result = append(result, tagDescription.LoadBalancerName)
				}
			}
		}

		if len(result) == 0 {
			return nil
		}

		log.Infoln("There are", len(result), "ELBs left from cluster", action.context.ClusterName)
		time.Sleep(10 * time.Second)
	}
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
