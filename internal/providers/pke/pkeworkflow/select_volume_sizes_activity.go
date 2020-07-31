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
	SelectVolumeSizesActivityName = "pke-get-ami-image-details-activity"
	MinimalVolumeSize             = 50 // TODO make this a config
)

type SelectVolumeSizesActivity struct {
	awsClientFactory *AWSClientFactory
}

type SelectVolumeSizesActivityInput struct {
	AWSActivityInput
	NodePools []NodePool
}

type AMIImageDetails struct {
	VolumeSize int
}

func NewSelectVolumeSizesActivity(awsClientFactory *AWSClientFactory) *SelectVolumeSizesActivity {
	return &SelectVolumeSizesActivity{
		awsClientFactory: awsClientFactory,
	}
}

func (a *SelectVolumeSizesActivity) Execute(ctx context.Context, input SelectVolumeSizesActivityInput) ([]NodePool, error) {
	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, err
	}

	e := ec2.New(client)

	nodePools := input.NodePools
	images := make(map[string]AMIImageDetails)
	for _, np := range nodePools {
		images[np.ImageID] = AMIImageDetails{}
	}

	var imageList []string
	for image := range images {
		imageList = append(imageList, image)
	}

	result, err := e.DescribeImages(&ec2.DescribeImagesInput{ImageIds: aws.StringSlice(imageList)})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to check AMI images")
	}

	sizes := make(map[string]int)

	for _, image := range result.Images {
		if len(image.BlockDeviceMappings) == 0 {
			err = errors.Combine(err, errors.NewWithDetails("AMI image has no block device mappings", "image", image.ImageId))
			continue
		}

		sizes[*image.ImageId] = int(*image.BlockDeviceMappings[0].Ebs.VolumeSize)
	}

	if err != nil {
		return nil, err
	}

	for i, pool := range nodePools {
		size, ok := sizes[pool.ImageID]
		if !ok {
			err = errors.Combine(err, errors.Errorf("AMI image %q for %q couldn't be found", pool.ImageID, pool.Name))
		}

		if pool.VolumeSize == 0 {
			nodePools[i].VolumeSize = size
			if size < MinimalVolumeSize {
				nodePools[i].VolumeSize = MinimalVolumeSize
			}
		} else {
			if pool.VolumeSize < size {
				err = errors.Combine(err, errors.Errorf("specified volume size of %dGB for %q is less than the AMI image size of %dGB", pool.VolumeSize, pool.Name, size))
			}
		}
	}

	return nodePools, nil
}
