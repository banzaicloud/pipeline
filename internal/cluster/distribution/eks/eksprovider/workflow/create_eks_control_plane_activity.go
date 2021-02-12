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
	"time"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/eks"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const CreateEksControlPlaneActivityName = "eks-create-control-plane"

// CreateEksControlPlaneActivity responsible for creating EKS control plane
type CreateEksControlPlaneActivity struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
}

// CreateEksControlPlaneActivityInput holds data needed for setting up EKS control plane
type CreateEksControlPlaneActivityInput struct {
	EKSActivityInput

	KubernetesVersion     string
	EncryptionConfig      []EncryptionConfig
	EndpointPrivateAccess bool
	EndpointPublicAccess  bool
	ClusterRoleArn        string
	SecurityGroupID       string
	LogTypes              []string
	Subnets               []Subnet
	Tags                  map[string]string
}

// CreateEksControlPlaneActivityOutput holds the output data of the CreateEksControlPlaneActivityOutput
type CreateEksControlPlaneActivityOutput struct {
}

// CreateEksControlPlaneActivity instantiates a new CreateEksControlPlaneActivity
func NewCreateEksClusterActivity(awsSessionFactory *awsworkflow.AWSSessionFactory) *CreateEksControlPlaneActivity {
	return &CreateEksControlPlaneActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *CreateEksControlPlaneActivity) Execute(ctx context.Context, input CreateEksControlPlaneActivityInput) (*CreateEksControlPlaneActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"region", input.Region,
		"version", input.KubernetesVersion,
	)

	outParams := CreateEksControlPlaneActivityOutput{}

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	eksSvc := eks.New(
		session,
		aws.NewConfig().
			WithLogger(aws.LoggerFunc(
				func(args ...interface{}) {
					logger.Debug(args)
				})).
			WithLogLevel(aws.LogDebugWithHTTPBody),
	)

	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(input.ClusterName),
	}

	createCluster := false

	describeClusterOutput, err := eksSvc.DescribeCluster(describeClusterInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == eks.ErrCodeResourceNotFoundException {
				createCluster = true
			} else {
				// sometimes error is different then ErrCodeResourceNotFoundException and Cluster is nil in describeClusterOutput
				if describeClusterOutput.Cluster == nil {
					return nil, errors.WrapIff(err, "could not get the status of EKS cluster %s in region %s", input.ClusterName, input.Region)
				}
				switch clusterStatus := aws.StringValue(describeClusterOutput.Cluster.Status); clusterStatus {
				case eks.ClusterStatusActive:
					logger.Infof("EKS cluster is in %s state", clusterStatus)
					return &outParams, nil
				case eks.ClusterStatusDeleting, eks.ClusterStatusFailed:
					return nil, errors.Errorf("EKS cluster with name %s already exists in region %s, state=%s", input.ClusterName, input.Region, clusterStatus)
				default:
					logger.Infof("EKS cluster is in %s state", clusterStatus)
				}
			}
		} else {
			return nil, errors.WrapIff(err, "could not get the status of EKS cluster %s in region %s", input.ClusterName, input.Region)
		}
	}

	if createCluster {
		var subnetIds []*string
		for i := range input.Subnets {
			subnetIds = append(subnetIds, &input.Subnets[i].SubnetID)
		}

		vpcConfigRequest := &eks.VpcConfigRequest{
			SecurityGroupIds:      []*string{&input.SecurityGroupID},
			SubnetIds:             subnetIds,
			EndpointPrivateAccess: aws.Bool(input.EndpointPrivateAccess),
			EndpointPublicAccess:  aws.Bool(input.EndpointPublicAccess),
		}

		roleArn := input.ClusterRoleArn

		logging := eks.Logging{
			ClusterLogging: []*eks.LogSetup{{
				Enabled: aws.Bool(true),
				Types:   aws.StringSlice(input.LogTypes),
			}},
		}

		tags := make(map[string]*string)
		for _, pipTag := range internalAmazon.PipelineTags() {
			tags[aws.StringValue(pipTag.Key)] = pipTag.Value
		}
		for k, v := range input.Tags {
			tags[k] = aws.String(v)
		}

		requestToken := aws.String(sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID))

		logger.Info("create EKS cluster")
		logger.Debug("clientRequestToken: ", requestToken)

		encryptionConfig := make([]*eks.EncryptionConfig, 0, len(input.EncryptionConfig))
		for encryptionConfigIndex := range input.EncryptionConfig {
			encryptionConfig = append(encryptionConfig, &eks.EncryptionConfig{
				Provider: &eks.Provider{
					KeyArn: aws.String(input.EncryptionConfig[encryptionConfigIndex].Provider.KeyARN),
				},
				Resources: aws.StringSlice(input.EncryptionConfig[encryptionConfigIndex].Resources),
			})
		}

		createClusterInput := &eks.CreateClusterInput{
			ClientRequestToken: requestToken,
			EncryptionConfig:   encryptionConfig,
			Name:               aws.String(input.ClusterName),
			ResourcesVpcConfig: vpcConfigRequest,
			RoleArn:            &roleArn,
			Logging:            &logging,
			Tags:               tags,
		}

		// set Kubernetes version only if provided, otherwise the cloud provider default one will be used
		if len(input.KubernetesVersion) > 0 {
			// EKS CreateCluster API accepts only major.minor Kubernetes version
			v, err := semver.NewVersion(input.KubernetesVersion)
			if err != nil {
				return nil, errors.WrapIff(err, "invalid Kubernetes version %q", input.KubernetesVersion)
			}

			createClusterInput.Version = aws.String(fmt.Sprintf("%d.%d", v.Major(), v.Minor()))
		}

		_, err = eksSvc.CreateCluster(createClusterInput)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to create EKS master")
		}
	}

	logger.Info("waiting for EKS cluster creation to finish")

	err = waitUntilClusterCreateCompleteWithContext(eksSvc, ctx, describeClusterInput)
	if err != nil {
		return nil, err
	}

	logger.Info("EKS cluster created successfully")

	return &outParams, nil
}

func waitUntilClusterCreateCompleteWithContext(eksSvc *eks.EKS, ctx aws.Context, input *eks.DescribeClusterInput, opts ...request.WaiterOption) error {
	// wait for 15 mins
	count := 0
	w := request.Waiter{
		Name:        "WaitUntilClusterCreateComplete",
		MaxAttempts: 30,
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
			count++
			activity.RecordHeartbeat(ctx, count)

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
