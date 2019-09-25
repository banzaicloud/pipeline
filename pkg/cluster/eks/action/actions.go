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

package action

import (
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
)

const awsNoUpdatesError = "No updates are to be performed."

// getStackTags returns the tags that are placed onto CF template stacks.
// These tags  are propagated onto the resources created by the CF template.
func getStackTags(clusterName, stackType string) []*cloudformation.Tag {
	return append([]*cloudformation.Tag{
		{Key: aws.String("banzaicloud-pipeline-cluster-name"), Value: aws.String(clusterName)},
		{Key: aws.String("banzaicloud-pipeline-stack-type"), Value: aws.String(stackType)},
	}, internalAmazon.PipelineTags()...)
}

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
	Subnets                    []*EksSubnet
	SSHKeyName                 string
	SSHKey                     *secret.SSHKeyPair
	VpcID                      *string
	VpcCidr                    *string
	ProvidedRoleArn            string
	APIEndpoint                *string
	CertificateAuthorityData   *string
	ClusterUserArn             string
	ClusterUserAccessKeyId     string
	ClusterUserSecretAccessKey string
	RouteTableID               *string
	ScaleEnabled               bool
}

// NewEksClusterCreationContext creates a new EksClusterCreateUpdateContext
func NewEksClusterCreationContext(session *session.Session, clusterName, sshKeyName string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
		SSHKeyName: sshKeyName,
	}
}

// NewEksClusterUpdateContext creates a new EksClusterCreateUpdateContext
func NewEksClusterUpdateContext(session *session.Session, clusterName string,
	securityGroupID *string, nodeSecurityGroupID *string, subnets []*EksSubnet, sshKeyName string, vpcID *string, nodeInstanceRoleId *string, clusterUserArn, clusterUserAccessKeyId, clusterUserSecretAccessKey string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
		SecurityGroupID:            securityGroupID,
		NodeSecurityGroupID:        nodeSecurityGroupID,
		Subnets:                    subnets,
		SSHKeyName:                 sshKeyName,
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
	VpcID            string
	SecurityGroupIDs []string
}

// NewEksClusterDeleteContext creates a new NewEksClusterDeleteContext
func NewEksClusterDeleteContext(session *session.Session, clusterName, vpcID string, securityGroupIDs []string) *EksClusterDeletionContext {
	return &EksClusterDeletionContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
		VpcID:            vpcID,
		SecurityGroupIDs: securityGroupIDs,
	}
}

// --

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
	// Store API endpoint, etc..
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	clusterInfo, err := eksSvc.DescribeCluster(describeClusterInput)
	if err != nil {
		return nil, err
	}
	cluster := clusterInfo.Cluster
	if cluster == nil {
		return nil, errors.New("unable to get EKS Cluster info")
	}

	a.context.APIEndpoint = cluster.Endpoint
	a.context.CertificateAuthorityData = cluster.CertificateAuthority.Data
	// TODO store settings in db

	return input, nil
}

// UndoAction rolls back this LoadEksSettingsAction
func (a *LoadEksSettingsAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO LoadEksSettingsAction")
	return nil
}

// --

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
				ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
				StackName:          aws.String(stackName),
			}
			_, err = cloudformationSrv.DeleteStack(deleteStackInput)
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() == cloudformation.ErrCodeStackInstanceNotFoundException {
						errorChan <- nil
						return
					}
				}
				errorChan <- err
				return
			}

			describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
			err = cloudformationSrv.WaitUntilStackDeleteComplete(describeStacksInput)
			if err != nil {
				errorChan <- pkgCloudformation.NewAwsStackFailure(err, stackName, cloudformationSrv)
				return
			}

			errorChan <- nil
		}(stackName)
	}

	var errs []error

	// wait for goroutines to finish
	for i := 0; i < len(a.StackNames); i++ {
		errs = append(errs, <-errorChan)
	}

	return nil, errors.Combine(errs...)
}

// --

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

	eksSrv := eks.New(a.context.Session)
	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	_, err = eksSrv.DeleteCluster(deleteClusterInput)

	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == eks.ErrCodeResourceNotFoundException {
			return nil, nil
		}
	}

	// wait until cluster exists
	startTime := time.Now()
	a.log.Info("waiting for EKS cluster deletion")
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	err = a.waitUntilClusterExists(aws.BackgroundContext(), describeClusterInput)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()
	a.log.Info("EKS cluster deleted successfully in", endTime.Sub(startTime).String())

	return nil, err
}

func (a *DeleteClusterAction) waitUntilClusterExists(ctx aws.Context, input *eks.DescribeClusterInput, opts ...request.WaiterOption) error {
	eksSvc := eks.New(a.context.Session)

	w := request.Waiter{
		Name:        "WaitUntilClusterExists",
		MaxAttempts: 30,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:    request.SuccessWaiterState,
				Matcher:  request.StatusWaiterMatch,
				Expected: 404,
			},
			{
				State:    request.RetryWaiterState,
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

// --

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

		var loadBalancerNames []*string
		describeLoadBalancers := &elb.DescribeLoadBalancersInput{}

		for {
			loadBalancers, err := elbService.DescribeLoadBalancers(describeLoadBalancers)
			if err != nil {
				return errors.WrapIf(err, "couldn't describe ELBs")
			}

			for _, description := range loadBalancers.LoadBalancerDescriptions {
				loadBalancerNames = append(loadBalancerNames, description.LoadBalancerName)
			}

			describeLoadBalancers.Marker = loadBalancers.NextMarker
			if loadBalancers.NextMarker == nil {
				break
			}
		}

		if len(loadBalancerNames) == 0 {
			return nil
		}

		// according to https://docs.aws.amazon.com/elasticloadbalancing/2012-06-01/APIReference/API_DescribeTags.html
		// tags can be queried for up to 20 ELBs in one call

		var result []*string
		maxELBNames := 20
		for low := 0; low < len(loadBalancerNames); low += maxELBNames {
			high := low + maxELBNames

			if high > len(loadBalancerNames) {
				high = len(loadBalancerNames)
			}

			describeTagsInput := &elb.DescribeTagsInput{
				LoadBalancerNames: loadBalancerNames[low:high],
			}

			describeTagsOutput, err := elbService.DescribeTags(describeTagsInput)
			if err != nil {
				return errors.WrapIf(err, "couldn't describe ELB tags")
			}

			for _, tagDescription := range describeTagsOutput.TagDescriptions {
				for _, tag := range tagDescription.Tags {
					if aws.StringValue(tag.Key) == clusterTag {
						result = append(result, tagDescription.LoadBalancerName)
					}
				}
			}
		}

		if len(result) == 0 {
			return nil
		}

		a.log.Infoln("there are", len(result), "ELBs left from cluster")
		time.Sleep(10 * time.Second)
	}
}
