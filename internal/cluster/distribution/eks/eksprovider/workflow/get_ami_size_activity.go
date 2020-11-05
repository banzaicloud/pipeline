// Copyright © 2020 Banzai Cloud
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

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const (
	GetAMISizeActivityName = "eks-get-ami-size-activity"
)

type GetAMISizeActivity struct {
	awsFactory awsworkflow.AWSFactory
	ec2Factory EC2APIFactory
}

type GetAMISizeActivityInput struct {
	EKSActivityInput
	ImageID string
}

type GetAMISizeActivityOutput struct {
	AMISize int
}

func NewGetAMISizeActivity(awsFactory awsworkflow.AWSFactory, ec2Factory EC2APIFactory) *GetAMISizeActivity {
	return &GetAMISizeActivity{
		awsFactory: awsFactory,
		ec2Factory: ec2Factory,
	}
}

func (a *GetAMISizeActivity) Execute(ctx context.Context, input GetAMISizeActivityInput) (*GetAMISizeActivityOutput, error) {
	awsClient, err := a.awsFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, err
	}

	ec2Client := a.ec2Factory.New(awsClient)

	describeImagesInput := ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(input.ImageID),
		},
	}
	result, err := ec2Client.DescribeImages(&describeImagesInput)
	if err != nil {
		return nil, errors.WrapIf(err, "describing AMI failed")
	} else if len(result.Images) == 0 {
		return nil, errors.NewWithDetails("describing AMI found no record", "image", input.ImageID)
	} else if len(result.Images[0].BlockDeviceMappings) == 0 {
		return nil, errors.NewWithDetails("describing AMI found no block device mappings", "image", input.ImageID)
	}

	return &GetAMISizeActivityOutput{
		AMISize: int(aws.Int64Value(result.Images[0].BlockDeviceMappings[0].Ebs.VolumeSize)),
	}, nil
}

// Register registers the activity.
func (a GetAMISizeActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: GetAMISizeActivityName})
}

// getAMISize retrieves the AMI size of the specified image ID using the
// provided values.
//
// This is a convenience wrapper around the corresponding activity.
func getAMISize(ctx workflow.Context, eksActivityInput EKSActivityInput, imageID string) (int, error) {
	var activityOutput GetAMISizeActivityOutput
	err := getAMISizeAsync(ctx, eksActivityInput, imageID).Get(ctx, &activityOutput)
	if err != nil {
		return 0, err
	}

	return activityOutput.AMISize, nil
}

// getAMISizeAsync returns a future object for retrieving the AMI size of the
// specified image ID using the provided values.
//
// This is a convenience wrapper around the corresponding activity.
func getAMISizeAsync(ctx workflow.Context, eksActivityInput EKSActivityInput, imageID string) workflow.Future {
	return workflow.ExecuteActivity(ctx, GetAMISizeActivityName, GetAMISizeActivityInput{
		EKSActivityInput: eksActivityInput,
		ImageID:          imageID,
	})
}
