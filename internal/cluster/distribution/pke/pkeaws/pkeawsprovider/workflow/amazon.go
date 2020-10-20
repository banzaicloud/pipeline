// Copyright © 2019 Banzai Cloud
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

package workflow

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
	zapadapter "logur.dev/adapter/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon/autoscaling"
	"github.com/banzaicloud/pipeline/src/secret"
)

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

const (
	asgWaitLoopSleep           = 5 * time.Second
	asgFulfillmentTimeout      = 2 * time.Minute
	asgFulfillmentWaitAttempts = asgFulfillmentTimeout / asgWaitLoopSleep
	asgFulfillmentWaitInterval = asgWaitLoopSleep
)

func GenerateStackNameForCluster(clusterName string) string {
	return "pipeline-eks-" + clusterName
}

func GenerateSSHKeyNameForCluster(clusterName string) string {
	return "pipeline-eks-ssh-" + clusterName
}

func GenerateNodePoolStackName(clusterName string, poolName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + poolName
}

// EKSActivityInput holds common input data for all activities
type EKSActivityInput struct {
	OrganizationID uint
	SecretID       string

	Region string

	ClusterName string

	// 64 chars length unique unique identifier that identifies the create CloudFormation
	AWSClientRequestTokenBase string
}

type EncryptionConfig struct {
	Provider  Provider
	Resources []string
}

type Provider struct {
	KeyARN string
}

// Subnet holds the fields of a Amazon subnet
type Subnet struct {
	SubnetID         string
	Cidr             string
	AvailabilityZone string
}

type AutoscaleGroup struct {
	Name             string
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeVolumeSize   int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string
	Delete           bool
	Create           bool
	CreatedBy        uint
}

type SecretStore interface {
	Get(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	GetByName(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
	Delete(organizationID uint, secretID string) error
	Update(organizationID uint, secretID string, request *secret.CreateSecretRequest) error
}

type Clusters interface {
	GetCluster(ctx context.Context, id uint) (EksCluster, error)
}

type EksCluster interface {
	GetModel() *eksmodel.EKSClusterModel
	Persist() error
	SetStatus(string, string) error
	DeleteFromDatabase() error
	GetConfigSecretId() string
	SaveConfigSecretId(secretID string) error
}

func WaitUntilStackCreateCompleteWithContext(cf *cloudformation.CloudFormation, ctx aws.Context, input *cloudformation.DescribeStacksInput, opts ...request.WaiterOption) error {
	count := 0
	w := request.Waiter{
		Name:        "WaitUntilStackCreateComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "CREATE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "CREATE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "ROLLBACK_COMPLETE",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		Logger: cf.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			count++
			activity.RecordHeartbeat(ctx, count)

			var inCpy *cloudformation.DescribeStacksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := cf.DescribeStacksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}

func WaitUntilStackUpdateCompleteWithContext(cf *cloudformation.CloudFormation, ctx aws.Context, input *cloudformation.DescribeStacksInput, opts ...request.WaiterOption) error {
	count := 0
	w := request.Waiter{
		Name:        "WaitUntilStackUpdateComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_COMPLETE",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		Logger: cf.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			count++
			activity.RecordHeartbeat(ctx, count)

			var inCpy *cloudformation.DescribeStacksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := cf.DescribeStacksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}

func WaitUntilStackDeleteCompleteWithContext(cf *cloudformation.CloudFormation, ctx aws.Context, input *cloudformation.DescribeStacksInput, opts ...request.WaiterOption) error {
	count := 0
	w := request.Waiter{
		Name:        "WaitUntilStackDeleteComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_COMPLETE",
			},
			{
				State:    request.SuccessWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "CREATE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_IN_PROGRESS",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_COMPLETE",
			},
		},
		Logger: cf.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			count++
			activity.RecordHeartbeat(ctx, count)

			var inCpy *cloudformation.DescribeStacksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := cf.DescribeStacksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}

// WaitForASGToBeFulfilled waits until an ASG has the desired amount of healthy nodes
func WaitForASGToBeFulfilled(
	ctx context.Context,
	logger *zap.SugaredLogger,
	awsSession *session.Session,
	stackName string,
	nodePoolName string) error {
	logger = logger.With("stackName", stackName)
	logger.Info("wait for ASG to be fulfilled")

	m := autoscaling.NewManager(awsSession, autoscaling.MetricsEnabled(true), autoscaling.Logger{
		Logger: zapadapter.New(logger.Desugar()),
	})

	ticker := time.NewTicker(asgFulfillmentWaitInterval)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-ticker.C:
			if i <= int(asgFulfillmentWaitAttempts) {
				i++
				activity.RecordHeartbeat(ctx, i)

				asGroup, err := m.GetAutoscalingGroupByStackName(stackName)
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						if aerr.Code() == "ValidationError" || aerr.Code() == "ASGNotFoundInResponse" {
							continue
						}
					}
					return errors.WrapIfWithDetails(err, "could not get ASG", "stackName", stackName)
				}

				ok, err := asGroup.IsHealthy()
				if err != nil {
					if autoscaling.IsErrorFinal(err) {
						return errors.WithDetails(err, "nodePoolName", nodePoolName, "stackName", aws.StringValue(asGroup.AutoScalingGroupName))
					}
					// log.Debug(err)
					continue
				}
				if ok {
					// log.Debug("ASG is healthy")
					return nil
				}
			} else {
				return errors.Errorf("waiting for ASG to be fulfilled timed out after %d x %s", asgFulfillmentWaitAttempts, asgFulfillmentWaitInterval)
			}
		case <-ctx.Done(): // wait for ASG fulfillment cancelled
			return nil
		}
	}
}
