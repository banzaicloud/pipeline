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

package cluster

import (
	"github.com/aws/aws-sdk-go/service/ec2"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

func newEC2Client(orgID uint, secretID string, region string) (*ec2.EC2, error) {
	s, err := getSecret(orgID, secretID)
	if err != nil {
		return nil, err
	}

	err = secret.ValidateSecretType(s, pkgCluster.Amazon)
	if err != nil {
		return nil, err
	}

	creds := verify.CreateAWSCredentials(s.Values)

	return verify.CreateEC2Client(creds, region)
}

// ListRegions lists supported regions
func ListRegions(orgId uint, secretId string, region string) ([]*ec2.Region, error) {
	client, err := newEC2Client(orgId, secretId, region)
	if err != nil {
		return nil, err
	}

	result, err := client.DescribeRegions(nil)
	if err != nil {
		return nil, err
	}

	return result.Regions, nil
}

// ListAMIs returns supported AMIs by region and tags
func ListAMIs(orgId uint, secretId string, region string, tags []*string) ([]*ec2.Image, error) {
	client, err := newEC2Client(orgId, secretId, region)
	if err != nil {
		return nil, err
	}

	var input *ec2.DescribeImagesInput
	if tags != nil {
		tagKey := "tag:Name"
		input = &ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{
					Name:   &tagKey,
					Values: tags,
				},
			},
		}
	}

	result, err := client.DescribeImages(input)
	if err != nil {
		return nil, err
	}

	return result.Images, nil
}
