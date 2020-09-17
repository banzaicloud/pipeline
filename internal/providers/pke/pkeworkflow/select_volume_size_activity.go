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

package pkeworkflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	SelectVolumeSizeActivityName = "pke-select-volume-size-activity"
)

type SelectVolumeSizeActivity struct {
	awsFactory AWSFactory
	ec2Factory EC2APIFactory
}

type SelectVolumeSizeActivityInput struct {
	AWSActivityInput
	ImageID string
}

type SelectVolumeSizeActivityOutput struct {
	VolumeSize int
}

func NewSelectVolumeSizeActivity(awsFactory AWSFactory, ec2Factory EC2APIFactory) *SelectVolumeSizeActivity {
	return &SelectVolumeSizeActivity{
		awsFactory: awsFactory,
		ec2Factory: ec2Factory,
	}
}

func (a *SelectVolumeSizeActivity) Execute(ctx context.Context, input SelectVolumeSizeActivityInput) (*SelectVolumeSizeActivityOutput, error) {
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
		return nil, errors.WrapIf(err, "describing AMI image failed")
	} else if len(result.Images) == 0 {
		return nil, errors.NewWithDetails("describing AMI image found no record", "image", input.ImageID)
	} else if len(result.Images[0].BlockDeviceMappings) == 0 {
		return nil, errors.NewWithDetails("describing AMI image found no block device mappings", "image", input.ImageID)
	}

	return &SelectVolumeSizeActivityOutput{
		VolumeSize: int(aws.Int64Value(result.Images[0].BlockDeviceMappings[0].Ebs.VolumeSize)),
	}, nil
}
