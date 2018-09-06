package action

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/banzaicloud/pipeline/model"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const awsNoUpdatesError = "No updates are to be performed."

// EksClusterContext describes the common fields used across EKS cluster create/update/delete operations
type EksClusterContext struct {
	Session     *session.Session
	ClusterName string
}

// EksClusterCreateUpdateContext describes the properties of an EKS cluster creation
type EksClusterCreateUpdateContext struct {
	EksClusterContext
	ClusterRoleArn             string
	NodeInstanceRoleID         *string
	NodeInstanceRoleArn        string
	SecurityGroupID            *string
	NodeSecurityGroupID        *string
	SubnetIDs                  []*string
	SSHKeyName                 string
	SSHKey                     *secret.SSHKeyPair
	VpcID                      *string
	ProvidedRoleArn            string
	APIEndpoint                *string
	CertificateAuthorityData   *string
	NodePoolTemplate           string
	ClusterUserArn             string
	ClusterUserAccessKeyId     string
	ClusterUserSecretAccessKey string
}

// NewEksClusterCreationContext creates a new EksClusterCreateUpdateContext
func NewEksClusterCreationContext(session *session.Session, clusterName, sshKeyName, nodePoolTemplate string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
		SSHKeyName:       sshKeyName,
		NodePoolTemplate: nodePoolTemplate,
	}
}

// NewEksClusterUpdateContext creates a new EksClusterCreateUpdateContext
func NewEksClusterUpdateContext(session *session.Session, clusterName string,
	securityGroupID *string, nodeSecurityGroupID *string, subnetIDs []*string, sshKeyName, nodePoolTemplate string, vpcID *string, nodeInstanceRoleId *string, clusterUserArn, clusterUserAccessKeyId, clusterUserSecretAccessKey string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
		SecurityGroupID:            securityGroupID,
		NodeSecurityGroupID:        nodeSecurityGroupID,
		SubnetIDs:                  subnetIDs,
		SSHKeyName:                 sshKeyName,
		NodePoolTemplate:           nodePoolTemplate,
		VpcID:                      vpcID,
		NodeInstanceRoleID:         nodeInstanceRoleId,
		ClusterUserArn:             clusterUserArn,
		ClusterUserAccessKeyId:     clusterUserAccessKeyId,
		ClusterUserSecretAccessKey: clusterUserSecretAccessKey,
	}
}

// EksClusterDeletionContext describes the properties of an EKS cluster deletion
type EksClusterDeletionContext struct {
	EksClusterContext
}

// NewEksClusterDeleteContext creates a new NewEksClusterDeleteContext
func NewEksClusterDeleteContext(session *session.Session, clusterName string) *EksClusterDeletionContext {
	return &EksClusterDeletionContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
	}
}

// --

var _ utils.RevocableAction = (*CreateVPCAndRolesAction)(nil)

// CreateVPCAndRolesAction describes the properties of a VPC creation
type CreateVPCAndRolesAction struct {
	context   *EksClusterCreateUpdateContext
	stackName string
	log       logrus.FieldLogger
}

// NewCreateVPCAndRolesAction creates a new CreateVPCAndRolesAction
func NewCreateVPCAndRolesAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, stackName string) *CreateVPCAndRolesAction {
	return &CreateVPCAndRolesAction{
		context:   creationContext,
		stackName: stackName,
		log:       log,
	}
}

// GetName returns the name of this CreateVPCAndRolesAction
func (a *CreateVPCAndRolesAction) GetName() string {
	return "CreateVPCAndRolesAction"
}

// ExecuteAction executes this CreateVPCAndRolesAction
func (a *CreateVPCAndRolesAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE CreateVPCAndRolesAction, stack name:", a.stackName)

	a.log.Infoln("Getting CloudFormation template for creating VPC for EKS cluster")
	templateBody, err := pkgEks.GetVPCTemplate()
	if err != nil {
		a.log.Errorln("Getting CloudFormation template for VPC failed:", err.Error())
		return nil, err
	}

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(a.context.ClusterName),
		},
	}

	cloudformationSrv := cloudformation.New(a.context.Session)

	createStackInput := &cloudformation.CreateStackInput{
		//Capabilities:       []*string{},
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		DisableRollback:    aws.Bool(false),
		Capabilities: []*string{
			aws.String(cloudformation.CapabilityCapabilityIam),
			aws.String(cloudformation.CapabilityCapabilityNamedIam),
		},
		StackName:        aws.String(a.stackName),
		Parameters:       stackParams,
		Tags:             []*cloudformation.Tag{{Key: aws.String("pipeline-created"), Value: aws.String("true")}},
		TemplateBody:     aws.String(templateBody),
		TimeoutInMinutes: aws.Int64(10),
	}
	_, err = cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		return
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(a.stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)
	if err != nil {
		logFailedStackEvents(a.log, a.stackName, cloudformationSrv)
	}
	return nil, err
}

func logFailedStackEvents(log logrus.FieldLogger, stackName string, cloudformationSrv *cloudformation.CloudFormation) {
	describeStackEventsInput := &cloudformation.DescribeStackEventsInput{StackName: aws.String(stackName)}
	describeStackEventsOutput, _ := cloudformationSrv.DescribeStackEvents(describeStackEventsInput)
	for _, event := range describeStackEventsOutput.StackEvents {
		if strings.HasSuffix(*event.ResourceStatus, "FAILED") {
			log.Errorf("stack %v event %v %v %v", aws.String(stackName), aws.StringValue(event.LogicalResourceId), aws.StringValue(event.ResourceStatus), aws.StringValue(event.ResourceStatusReason))
		}
	}
}

// UndoAction rolls back this CreateVPCAndRolesAction
func (a *CreateVPCAndRolesAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO CreateVPCAndRolesAction, deleting stack:", a.stackName)
	cloudformationSrv := cloudformation.New(a.context.Session)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		StackName:          aws.String(a.stackName),
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
	log       logrus.FieldLogger
}

// NewGenerateVPCConfigRequestAction creates a new GenerateVPCConfigRequestAction
func NewGenerateVPCConfigRequestAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, stackName string) *GenerateVPCConfigRequestAction {
	return &GenerateVPCConfigRequestAction{
		context:   creationContext,
		stackName: stackName,
		log:       log,
	}
}

// GetName returns the name of this GenerateVPCConfigRequestAction
func (a *GenerateVPCConfigRequestAction) GetName() string {
	return "GenerateVPCConfigRequestAction"
}

// ExecuteAction executes this GenerateVPCConfigRequestAction
func (a *GenerateVPCConfigRequestAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE GenerateVPCConfigRequestAction, stack name:", a.stackName)
	cloudformationSrv := cloudformation.New(a.context.Session)

	describeStackResourcesInput := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(a.stackName),
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
	nodeSecurityGroup, found := stackResourceMap["NodeSecurityGroup"]
	if !found {
		return nil, errors.New("Unable to find NodeSecurityGroup resource")
	}
	subnet01resource, found := stackResourceMap["Subnet01"]
	if !found {
		return nil, errors.New("Unable to find Subnet02 resource")
	}
	subnet02resource, found := stackResourceMap["Subnet02"]
	if !found {
		return nil, errors.New("Unable to find Subnet01 resource")
	}
	vpcResource, found := stackResourceMap["VPC"]
	if !found {
		return nil, errors.New("Unable to find VPC resource")
	}
	nodeInstanceProfileResource, found := stackResourceMap["NodeInstanceRole"]
	if !found {
		return nil, errors.New("Unable to find NodeInstanceRole resource")
	}

	a.log.Infof("Stack resources: %v", stackResources)

	a.context.VpcID = vpcResource.PhysicalResourceId
	a.context.SecurityGroupID = securityGroupResource.PhysicalResourceId
	a.context.SubnetIDs = []*string{subnet01resource.PhysicalResourceId, subnet02resource.PhysicalResourceId}
	a.context.NodeInstanceRoleID = nodeInstanceProfileResource.PhysicalResourceId
	a.context.NodeSecurityGroupID = nodeSecurityGroup.PhysicalResourceId

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(a.stackName)}
	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.New("Unable to find stack " + a.stackName)
	}

	var clusterRoleArn, nodeInstanceRoleArn, clusterUserArn, clusterUserAccessKeyId, clusterUserSecretAccessKey string
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "ClusterRoleArn":
			clusterRoleArn = aws.StringValue(output.OutputValue)
		case "NodeInstanceRoleArn":
			nodeInstanceRoleArn = aws.StringValue(output.OutputValue)
		case "ClusterUserArn":
			clusterUserArn = aws.StringValue(output.OutputValue)
		case "ClusterUserAccessKeyId":
			clusterUserAccessKeyId = aws.StringValue(output.OutputValue)
		case "ClusterUserSecretAccessKey":
			clusterUserSecretAccessKey = aws.StringValue(output.OutputValue)
		}
	}
	a.log.Infof("cluster role ARN: %v", clusterRoleArn)
	a.context.ClusterRoleArn = clusterRoleArn

	a.log.Infof("nodeInstanceRoleArn role ARN: %v", nodeInstanceRoleArn)
	a.context.NodeInstanceRoleArn = nodeInstanceRoleArn

	a.log.Infof("cluster user ARN: %v", clusterUserArn)
	a.context.ClusterUserArn = clusterUserArn

	a.log.Infof("cluster user access key id: %v", clusterUserAccessKeyId)
	a.context.ClusterUserAccessKeyId = clusterUserAccessKeyId
	a.context.ClusterUserSecretAccessKey = clusterUserSecretAccessKey

	return &eks.VpcConfigRequest{
		SecurityGroupIds: []*string{a.context.SecurityGroupID},
		SubnetIds:        a.context.SubnetIDs,
	}, nil
}

// UndoAction rolls back this GenerateVPCConfigRequestAction
func (a *GenerateVPCConfigRequestAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO GenerateVPCConfigRequestAction, stack name:", a.stackName)
	return nil
}

// --

var _ utils.RevocableAction = (*CreateEksClusterAction)(nil)

// CreateEksClusterAction describes the properties of an EKS cluster creation
type CreateEksClusterAction struct {
	context           *EksClusterCreateUpdateContext
	kubernetesVersion string
	log               logrus.FieldLogger
}

// NewCreateEksClusterAction creates a new CreateEksClusterAction
func NewCreateEksClusterAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, kubernetesVersion string) *CreateEksClusterAction {
	return &CreateEksClusterAction{
		context:           creationContext,
		kubernetesVersion: kubernetesVersion,
		log:               log,
	}
}

// GetName returns the name of this CreateEksClusterAction
func (a *CreateEksClusterAction) GetName() string {
	return "CreateEksClusterAction"
}

// ExecuteAction executes this CreateEksClusterAction
func (a *CreateEksClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	vpcConfigRequest, ok := input.(*eks.VpcConfigRequest)

	if !ok {
		return nil, errors.New("input parameter must be a *VpcConfigRequest")
	}
	a.log.Infoln("EXECUTE CreateEksClusterAction, cluster name")
	eksSvc := eks.New(a.context.Session)

	roleArn := a.context.ClusterRoleArn

	createClusterInput := &eks.CreateClusterInput{
		ClientRequestToken: aws.String(uuid.NewV4().String()),
		Name:               aws.String(a.context.ClusterName),
		ResourcesVpcConfig: vpcConfigRequest,
		RoleArn:            &roleArn,
	}

	// set Kubernetes version only if provided, otherwise the cloud provider default one will be used
	if len(a.kubernetesVersion) > 0 {
		createClusterInput.Version = aws.String(a.kubernetesVersion)
	}

	result, err := eksSvc.CreateCluster(createClusterInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			a.log.Errorf("CreateCluster error [%s]: %s", aerr.Code(), aerr.Error())
		} else {
			a.log.Errorf("CreateCluster error: %s", err.Error())
		}
		return nil, err
	}

	//wait for ready status
	startTime := time.Now()
	a.log.Info("Waiting for EKS cluster creation")
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	err = a.waitUntilClusterCreateComplete(describeClusterInput)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()
	a.log.Infoln("EKS cluster created successfully in", endTime.Sub(startTime).String())

	return result.Cluster, nil
}

func (a *CreateEksClusterAction) waitUntilClusterCreateComplete(input *eks.DescribeClusterInput) error {
	return a.waitUntilClusterCreateCompleteWithContext(aws.BackgroundContext(), input)
}

func (a *CreateEksClusterAction) waitUntilClusterCreateCompleteWithContext(ctx aws.Context, input *eks.DescribeClusterInput, opts ...request.WaiterOption) error {
	eksSvc := eks.New(a.context.Session)

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
func (a *CreateEksClusterAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO CreateEksClusterAction")
	eksSvc := eks.New(a.context.Session)

	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(a.context.ClusterName),
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
	log       logrus.FieldLogger
}

// NewCreateUpdateNodePoolStackAction creates a new CreateUpdateNodePoolStackAction
func NewCreateUpdateNodePoolStackAction(
	log logrus.FieldLogger,
	isCreate bool,
	creationContext *EksClusterCreateUpdateContext,
	nodePools ...*model.AmazonNodePoolsModel) *CreateUpdateNodePoolStackAction {
	return &CreateUpdateNodePoolStackAction{
		context:   creationContext,
		isCreate:  isCreate,
		nodePools: nodePools,
		log:       log,
	}
}

func (a *CreateUpdateNodePoolStackAction) generateStackName(nodePool *model.AmazonNodePoolsModel) string {
	return a.context.ClusterName + "-pipeline-eks-nodepool-" + nodePool.Name
}

// GetName return the name of this action
func (a *CreateUpdateNodePoolStackAction) GetName() string {
	return "CreateUpdateNodePoolStackAction"
}

// ExecuteAction executes the CreateUpdateNodePoolStackAction in parallel for each node pool
func (a *CreateUpdateNodePoolStackAction) ExecuteAction(input interface{}) (output interface{}, err error) {

	errorChan := make(chan error, len(a.nodePools))
	defer close(errorChan)

	for _, nodePool := range a.nodePools {

		go func(nodePool *model.AmazonNodePoolsModel) {

			stackName := a.generateStackName(nodePool)

			if a.isCreate {
				a.log.Infoln("EXECUTE CreateUpdateNodePoolStackAction, create stack name:", stackName)
			} else {
				a.log.Infoln("EXECUTE CreateUpdateNodePoolStackAction, update stack name:", stackName)
			}

			commaDelimitedSubnetIDs := ""
			for i, subnetID := range a.context.SubnetIDs {
				commaDelimitedSubnetIDs = commaDelimitedSubnetIDs + *subnetID
				if i != len(a.context.SubnetIDs)-1 {
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
					ParameterValue: aws.String(a.context.SSHKeyName),
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
					ParameterValue: aws.String(a.context.ClusterName),
				},
				{
					ParameterKey:   aws.String("NodeGroupName"),
					ParameterValue: aws.String(nodePool.Name),
				},
				{
					ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
					ParameterValue: a.context.SecurityGroupID,
				},
				{
					ParameterKey:   aws.String("NodeSecurityGroup"),
					ParameterValue: a.context.NodeSecurityGroupID,
				},
				{
					ParameterKey:   aws.String("VpcId"),
					ParameterValue: a.context.VpcID,
				}, {
					ParameterKey:   aws.String("Subnets"),
					ParameterValue: aws.String(commaDelimitedSubnetIDs),
				},
				{
					ParameterKey:   aws.String("NodeInstanceRoleId"),
					ParameterValue: a.context.NodeInstanceRoleID,
				},
				{
					ParameterKey:   aws.String("BootstrapArguments"),
					ParameterValue: aws.String(fmt.Sprintf("--kubelet-extra-args '--node-labels pipeline-nodepool-name=%v'", nodePool.Name)),
				},
			}

			cloudformationSrv := cloudformation.New(a.context.Session)

			waitOnCreateUpdate := true

			// create stack
			if a.isCreate {
				createStackInput := &cloudformation.CreateStackInput{
					ClientRequestToken: aws.String(uuid.NewV4().String()),
					DisableRollback:    aws.Bool(false),
					StackName:          aws.String(stackName),
					Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
					Parameters:         stackParams,
					Tags:               tags,
					TemplateBody:       aws.String(a.context.NodePoolTemplate),
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
						a.log.Warnf("Nothing changed during update!")
						waitOnCreateUpdate = false
						err = nil
					} else {
						errorChan <- err
						return
					}
				}
			}

			describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}

			if a.isCreate {
				err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)
			} else if waitOnCreateUpdate {
				err = cloudformationSrv.WaitUntilStackUpdateComplete(describeStacksInput)
			}

			if err != nil {
				logFailedStackEvents(a.log, stackName, cloudformationSrv)
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
	for i := 0; i < len(a.nodePools); i++ {
		createErr := <-errorChan
		if createErr != nil {
			err = createErr
		}
	}

	return nil, err
}

// UndoAction rolls back this CreateUpdateNodePoolStackAction
func (a *CreateUpdateNodePoolStackAction) UndoAction() (err error) {
	for _, nodepool := range a.nodePools {
		a.log.Info("EXECUTE UNDO CreateUpdateNodePoolStackAction")
		cloudformationSrv := cloudformation.New(a.context.Session)
		deleteStackInput := &cloudformation.DeleteStackInput{
			ClientRequestToken: aws.String(uuid.NewV4().String()),
			StackName:          aws.String(a.generateStackName(nodepool)),
		}
		_, deleteErr := cloudformationSrv.DeleteStack(deleteStackInput)
		if deleteErr != nil {
			a.log.Errorln("Error during deleting CloudFormation stack:", err.Error())
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
	log       logrus.FieldLogger
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(log logrus.FieldLogger, context *EksClusterCreateUpdateContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
	return &UploadSSHKeyAction{
		context:   context,
		sshSecret: sshSecret,
		log:       log,
	}
}

// GetName returns the name of this UploadSSHKeyAction
func (a *UploadSSHKeyAction) GetName() string {
	return "UploadSSHKeyAction"
}

// ExecuteAction executes this UploadSSHKeyAction
func (a *UploadSSHKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE UploadSSHKeyAction")

	a.context.SSHKey = secret.NewSSHKeyPair(a.sshSecret)
	ec2srv := ec2.New(a.context.Session)
	importKeyPairInput := &ec2.ImportKeyPairInput{
		// A unique name for the key pair.
		// KeyName is a required field
		KeyName: aws.String(a.context.SSHKeyName),

		// The public key. For API calls, the text must be base64-encoded. For command
		// line tools, base64 encoding is performed for you.
		//
		// PublicKeyMaterial is automatically base64 encoded/decoded by the SDK.
		//
		// PublicKeyMaterial is a required field
		PublicKeyMaterial: []byte(a.context.SSHKey.PublicKeyData), // []byte `locationName:"publicKeyMaterial" type:"blob" required:"true"`
	}
	output, err = ec2srv.ImportKeyPair(importKeyPairInput)
	return output, err
}

// UndoAction rolls back this UploadSSHKeyAction
func (a *UploadSSHKeyAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO UploadSSHKeyAction")
	//delete uploaded keypair
	ec2srv := ec2.New(a.context.Session)

	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(a.context.SSHKeyName),
	}
	_, err = ec2srv.DeleteKeyPair(deleteKeyPairInput)
	return err
}

// ---

var _ utils.RevocableAction = (*RevertStepsAction)(nil)

// RevertStepsAction can be used to intentionally revert all the steps (=simulate an error)
type RevertStepsAction struct {
	log logrus.FieldLogger
}

// NewRevertStepsAction creates a new RevertStepsAction
func NewRevertStepsAction(log logrus.FieldLogger) *RevertStepsAction {
	return &RevertStepsAction{log: log}
}

// GetName returns the name of this RevertStepsAction
func (a *RevertStepsAction) GetName() string {
	return "RevertStepsAction"
}

// ExecuteAction executes this RevertStepsAction
func (a *RevertStepsAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE RevertStepsAction")
	return nil, errors.New("Intentionally reverting everything")
}

// UndoAction rolls back this RevertStepsAction
func (a *RevertStepsAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO RevertStepsAction")
	return nil
}

// ---

var _ utils.RevocableAction = (*LoadEksSettingsAction)(nil)

// LoadEksSettingsAction to describe the EKS cluster created
type LoadEksSettingsAction struct {
	context *EksClusterCreateUpdateContext
	log     logrus.FieldLogger
}

// NewLoadEksSettingsAction creates a new LoadEksSettingsAction
func NewLoadEksSettingsAction(log logrus.FieldLogger, context *EksClusterCreateUpdateContext) *LoadEksSettingsAction {
	return &LoadEksSettingsAction{
		context: context,
		log:     log,
	}
}

// GetName returns the name of this LoadEksSettingsAction
func (a *LoadEksSettingsAction) GetName() string {
	return "LoadEksSettingsAction"
}

// ExecuteAction executes this LoadEksSettingsAction
func (a *LoadEksSettingsAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE LoadEksSettingsAction")
	eksSvc := eks.New(a.context.Session)
	//Store API endpoint, etc..
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	clusterInfo, err := eksSvc.DescribeCluster(describeClusterInput)
	if err != nil {
		return nil, err
	}
	cluster := clusterInfo.Cluster
	if cluster == nil {
		return nil, errors.New("Unable to get EKS Cluster info")
	}

	a.context.APIEndpoint = cluster.Endpoint
	a.context.CertificateAuthorityData = cluster.CertificateAuthority.Data
	//TODO store settings in db

	return input, nil
}

// UndoAction rolls back this LoadEksSettingsAction
func (a *LoadEksSettingsAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO LoadEksSettingsAction")
	return nil
}

//--

var _ utils.Action = (*DeleteStackAction)(nil)

// DeleteStackAction deletes a stack
type DeleteStackAction struct {
	context    *EksClusterDeletionContext
	StackNames []string
	log        logrus.FieldLogger
}

// NewDeleteStacksAction creates a new DeleteStackAction
func NewDeleteStacksAction(log logrus.FieldLogger, context *EksClusterDeletionContext, stackNames ...string) *DeleteStackAction {
	return &DeleteStackAction{
		context:    context,
		StackNames: stackNames,
		log:        log,
	}
}

// GetName returns the name of this DeleteStackAction
func (a *DeleteStackAction) GetName() string {
	return "DeleteStackAction"
}

// ExecuteAction executes this DeleteStackAction
func (a *DeleteStackAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infof("EXECUTE DeleteStackAction: %q", a.StackNames)

	errorChan := make(chan error, len(a.StackNames))
	defer close(errorChan)

	for _, stackName := range a.StackNames {
		go func(stackName string) {
			cloudformationSrv := cloudformation.New(a.context.Session)
			deleteStackInput := &cloudformation.DeleteStackInput{
				ClientRequestToken: aws.String(uuid.NewV4().String()),
				StackName:          aws.String(stackName),
			}
			_, err = cloudformationSrv.DeleteStack(deleteStackInput)
			if err != nil {
				errorChan <- err
				return
			}

			describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
			err = cloudformationSrv.WaitUntilStackDeleteComplete(describeStacksInput)
			if err != nil {
				logFailedStackEvents(a.log, stackName, cloudformationSrv)

				errorChan <- err
				return
			}

			errorChan <- nil
		}(stackName)
	}

	// wait for goroutines to finish
	for i := 0; i < len(a.StackNames); i++ {
		deleteErr := <-errorChan
		if deleteErr != nil {
			err = deleteErr
		}
	}

	return nil, err
}

//--

var _ utils.Action = (*DeleteClusterAction)(nil)

// DeleteClusterAction deletes an EKS cluster
type DeleteClusterAction struct {
	context *EksClusterDeletionContext
	log     logrus.FieldLogger
}

// NewDeleteClusterAction creates a new DeleteClusterAction
func NewDeleteClusterAction(log logrus.FieldLogger, context *EksClusterDeletionContext) *DeleteClusterAction {
	return &DeleteClusterAction{
		context: context,
		log:     log,
	}
}

// GetName returns the name of this DeleteClusterAction
func (a *DeleteClusterAction) GetName() string {
	return "DeleteClusterAction"
}

// ExecuteAction executes this DeleteClusterAction
func (a *DeleteClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteClusterAction")

	//TODO handle non existing cluster
	eksSrv := eks.New(a.context.Session)
	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	return eksSrv.DeleteCluster(deleteClusterInput)
}

//--

var _ utils.Action = (*DeleteSSHKeyAction)(nil)

// DeleteSSHKeyAction deletes a generated SSH key
type DeleteSSHKeyAction struct {
	context    *EksClusterDeletionContext
	SSHKeyName string
	log        logrus.FieldLogger
}

// NewDeleteSSHKeyAction creates a new DeleteSSHKeyAction
func NewDeleteSSHKeyAction(log logrus.FieldLogger, context *EksClusterDeletionContext, sshKeyName string) *DeleteSSHKeyAction {
	return &DeleteSSHKeyAction{
		context:    context,
		SSHKeyName: sshKeyName,
		log:        log,
	}
}

// GetName returns the name of this DeleteSSHKeyAction
func (a *DeleteSSHKeyAction) GetName() string {
	return "DeleteSSHKeyAction"
}

// ExecuteAction executes this DeleteSSHKeyAction
func (a *DeleteSSHKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteSSHKeyAction")

	//TODO handle non existing key
	ec2srv := ec2.New(a.context.Session)
	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(a.SSHKeyName),
	}
	return ec2srv.DeleteKeyPair(deleteKeyPairInput)
}

//--

var _ utils.Action = (*WaitResourceDeletionAction)(nil)

// WaitResourceDeletionAction deletes a generated SSH key
type WaitResourceDeletionAction struct {
	context *EksClusterDeletionContext
	log     logrus.FieldLogger
}

// NewWaitResourceDeletionAction creates a new WaitResourceDeletionAction
func NewWaitResourceDeletionAction(log logrus.FieldLogger, context *EksClusterDeletionContext) *WaitResourceDeletionAction {
	return &WaitResourceDeletionAction{
		context: context,
		log:     log,
	}
}

// GetName returns the name of this WaitResourceDeletionAction
func (a *WaitResourceDeletionAction) GetName() string {
	return "WaitResourceDeletionAction"
}

// ExecuteAction executes this WaitResourceDeletionAction
func (a *WaitResourceDeletionAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE WaitResourceDeletionAction")

	return nil, a.waitUntilELBsDeleted()
}

func (a *WaitResourceDeletionAction) waitUntilELBsDeleted() error {

	elbService := elb.New(a.context.Session)
	clusterTag := "kubernetes.io/cluster/" + a.context.ClusterName

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

		a.log.Infoln("There are", len(result), "ELBs left from cluster")
		time.Sleep(10 * time.Second)
	}
}
