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
	"fmt"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	zapadapter "logur.dev/adapter/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon/autoscaling"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

const (
	asgWaitLoopSleep           = 5 * time.Second
	asgFulfillmentTimeout      = 2 * time.Minute
	asgFulfillmentWaitAttempts = asgFulfillmentTimeout / asgWaitLoopSleep
	asgFulfillmentWaitInterval = asgWaitLoopSleep
)

// getStackTags returns the tags that are placed onto CF template stacks.
// These tags  are propagated onto the resources created by the CF template.
func getStackTags(clusterName, stackType string, customTagsMap map[string]string) []*cloudformation.Tag {
	tags := make([]*cloudformation.Tag, 0)

	for k, v := range customTagsMap {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	tags = append(tags, []*cloudformation.Tag{
		{Key: aws.String("banzaicloud-pipeline-cluster-name"), Value: aws.String(clusterName)},
		{Key: aws.String("banzaicloud-pipeline-stack-type"), Value: aws.String(stackType)},
	}...)
	tags = append(tags, internalAmazon.PipelineTags()...)
	return tags
}

func getNodePoolStackTags(clusterName string, customTagsMap map[string]string) []*cloudformation.Tag {
	return getStackTags(clusterName, "nodepool", customTagsMap)
}

func GenerateStackNameForCluster(clusterName string) string {
	return "pipeline-eks-" + clusterName
}

func generateStackNameForSubnet(clusterName, subnetCidr string) string {
	r := strings.NewReplacer(".", "-", "/", "-")
	return fmt.Sprintf("pipeline-eks-subnet-%s-%s", clusterName, r.Replace(subnetCidr))
}

func generateStackNameForIam(clusterName string) string {
	return "pipeline-eks-iam-" + clusterName
}

func GenerateSSHKeyNameForCluster(clusterName string) string {
	return "pipeline-eks-ssh-" + clusterName
}

func GenerateNodePoolStackName(clusterName string, poolName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + poolName
}

// getSecretName returns the name that identifies the  cluster user access key in Vault
func getSecretName(userName string) string {
	return fmt.Sprintf("%s-key", strings.ToLower(userName))
}

func generateK8sConfig(clusterName string, apiEndpoint string, certificateAuthorityData []byte,
	awsAccessKeyID string, awsSecretAccessKey string) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmdapi.Cluster{
					Server:                   apiEndpoint,
					CertificateAuthorityData: certificateAuthorityData,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: clusterName,
				Context: clientcmdapi.Context{
					AuthInfo: "eks",
					Cluster:  clusterName,
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
						Args:       []string{"token", "-i", clusterName},
						Env: []clientcmdapi.ExecEnvVar{
							{Name: "AWS_ACCESS_KEY_ID", Value: awsAccessKeyID},
							{Name: "AWS_SECRET_ACCESS_KEY", Value: awsSecretAccessKey},
						},
					},
				},
			},
		},
		Kind:           "Config",
		CurrentContext: clusterName,
	}
}

func packageCFError(err error, stackName string, clientRequestToken string, cloudformationClient *cloudformation.CloudFormation, errMessage string) error {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
			err = pkgCloudformation.NewAwsStackFailure(err, stackName, clientRequestToken, cloudformationClient)
			err = errors.WrapIfWithDetails(err, errMessage, "stackName", stackName)
			if pkgCloudformation.IsErrorFinal(err) {
				return cadence.NewCustomError(ErrReasonStackFailed, err.Error())
			}
			return err
		}
	}
	return err
}

// EKSActivityInput holds common input data for all activities
// Deprecated! Use the AWSCommonActivityInput from "github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow" instead
type EKSActivityInput struct {
	OrganizationID uint
	SecretID       string

	Region string

	ClusterName string
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

// NewSubnetsFromEKSSubnets returns subnet objects optionally matching the
// specified subnet IDs from a EKS subnet model collection or an error if a
// subnet ID is not found among the provided EKS subnet models. If no subnet IDs
// are specified then all EKS subnet models are returned as subnets.
func NewSubnetsFromEKSSubnets(
	eksSubnets []*eksmodel.EKSSubnetModel,
	optionalIncludedSubnetIDs ...string,
) ([]Subnet, error) {
	includedSubnetCount := len(optionalIncludedSubnetIDs)
	matchedSubnets := make([]Subnet, 0, includedSubnetCount)

	for _, eksSubnet := range eksSubnets {
		if includedSubnetCount == 0 ||
			indexStrings(optionalIncludedSubnetIDs, aws.StringValue(eksSubnet.SubnetId)) != -1 {
			matchedSubnets = append(matchedSubnets, Subnet{
				SubnetID:         aws.StringValue(eksSubnet.SubnetId),
				Cidr:             aws.StringValue(eksSubnet.Cidr),
				AvailabilityZone: aws.StringValue(eksSubnet.AvailabilityZone),
			})
		}
	}

	if includedSubnetCount != 0 &&
		len(matchedSubnets) != includedSubnetCount {
		return nil, errors.NewWithDetails(
			"some subnet IDs could not be found among the subnets",
			"subnetIds", optionalIncludedSubnetIDs,
			"subnets", eksSubnets,
		)
	}

	return matchedSubnets, nil
}

// TODO: remove when UpdateNodePoolWorkflow is refactored and this is not needed
// anymore.
type AutoscaleGroup struct {
	Name                 string
	NodeSpotPrice        string
	Autoscaling          bool
	NodeMinCount         int
	NodeMaxCount         int
	Count                int
	NodeVolumeEncryption *eks.NodePoolVolumeEncryption
	NodeVolumeSize       int
	NodeImage            string
	NodeInstanceType     string

	// SecurityGroups collects the user specified custom node security group
	// IDs.
	SecurityGroups   []string
	UseInstanceStore *bool

	Labels    map[string]string
	Delete    bool
	Create    bool
	CreatedBy uint
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
