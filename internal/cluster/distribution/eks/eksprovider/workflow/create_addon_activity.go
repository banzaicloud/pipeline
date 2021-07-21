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
	"github.com/aws/aws-sdk-go/service/eks"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

const CreateAddonActivityName = "eks-create-addon"

// CreateAddonActivity creates an EKS addon
type CreateAddonActivity struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
}

// CreateAddonActivityInput holds input data
type CreateAddonActivityInput struct {
	EKSActivityInput

	KubernetesVersion string
	AddonName         string
}

// CreateAddonActivityOutput holds the output data
type CreateAddonActivityOutput struct {
}

// NewCreateAddonActivity instantiates a new CreateAddonActivity
func NewCreateAddonActivity(awsSessionFactory *awsworkflow.AWSSessionFactory) *CreateAddonActivity {
	return &CreateAddonActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *CreateAddonActivity) Execute(ctx context.Context, input CreateAddonActivityInput) (*CreateAddonActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"region", input.Region,
		"version", input.KubernetesVersion,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}
	eksSvc := eks.New(
		awsSession,
		aws.NewConfig().
			WithLogger(aws.LoggerFunc(
				func(args ...interface{}) {
					logger.Debug(args)
				})).
			WithLogLevel(aws.LogDebugWithHTTPBody),
	)

	logger = logger.With("addonName", input.AddonName)
	logger.Info("create add-on for cluster: " + (input.ClusterName))
	t := time.Now().Unix()

	addOnInput := &eks.CreateAddonInput{
		AddonName:        aws.String(input.AddonName),
		ClusterName:      aws.String(input.ClusterName),
		ResolveConflicts: aws.String(eks.ResolveConflictsOverwrite),
	}
	addOnOutput, err := eksSvc.CreateAddon(addOnInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceInUseException" {
			logger.Info("addon created but: " + awsErr.Message())
		} else {
			return nil, errors.Wrap(err, "failed to create coredns add-on")
		}
	} else {
		logger.Info("addon created with status: " + addOnOutput.Addon.String())
	}

	logger.Info("waiting for add-on creation")

	describeAddonInput := &eks.DescribeAddonInput{
		AddonName:   aws.String(input.AddonName),
		ClusterName: aws.String(input.ClusterName),
	}
	err = waitUntilAddOnCreateCompleteWithContext(eksSvc, ctx, describeAddonInput)
	if err != nil {
		return nil, err
	}

	t = time.Now().Unix() - t
	logger.Infof("add-on created successfully in %v secs", t)

	outParams := CreateAddonActivityOutput{}
	return &outParams, nil
}

func waitUntilAddOnCreateCompleteWithContext(eksSvc *eks.EKS, ctx aws.Context, input *eks.DescribeAddonInput, opts ...request.WaiterOption) error {
	// wait for 15 mins
	count := 0
	w := request.Waiter{
		Name:        "WaitUntilAddOnCreateCompleteWithContext",
		MaxAttempts: 30,
		Delay:       request.ConstantWaiterDelay(10 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Addon.Status",
				Expected: eks.AddonStatusActive,
			},
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Addon.Status",
				Expected: eks.AddonStatusDegraded,
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Addon.Status",
				Expected: eks.AddonStatusDeleting,
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Addon.Status",
				Expected: eks.AddonStatusCreateFailed,
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

			var inCpy *eks.DescribeAddonInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := eksSvc.DescribeAddonRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}
