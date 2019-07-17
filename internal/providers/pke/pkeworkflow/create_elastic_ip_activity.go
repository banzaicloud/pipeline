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

package pkeworkflow

import (
	"context"

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/cadence/activity"
)

const CreateElasticIPActivityName = "pke-create-eip-activity"

type CreateElasticIPActivity struct {
	awsClientFactory *AWSClientFactory
}

func NewCreateElasticIPActivity(awsClientFactory *AWSClientFactory) *CreateElasticIPActivity {
	return &CreateElasticIPActivity{
		awsClientFactory: awsClientFactory,
	}
}

type CreateElasticIPActivityInput struct {
	AWSActivityInput
	ClusterID   uint
	ClusterName string
}

type CreateElasticIPActivityOutput struct {
	PublicIp     string
	AllocationId string
}

func (a *CreateElasticIPActivity) Execute(ctx context.Context, input CreateElasticIPActivityInput) (*CreateElasticIPActivityOutput, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)

	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, err
	}

	e := ec2.New(client)

	// check EIP is already allocated or not
	descAddrIn := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: aws.StringSlice([]string{"kubernetes.io/cluster/" + input.ClusterName}),
			},
		},
	}
	descAddrOut, err := e.DescribeAddresses(descAddrIn)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to query EIP based on tag")
	}

	if descAddrOut != nil && len(descAddrOut.Addresses) > 0 {
		log.Infof("EIP already exists: %s", *descAddrOut.Addresses[0].PublicIp)
		output := &CreateElasticIPActivityOutput{
			PublicIp:     *descAddrOut.Addresses[0].PublicIp,
			AllocationId: *descAddrOut.Addresses[0].AllocationId,
		}
		return output, nil
	}

	addrIn := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	}
	addrOut, err := e.AllocateAddress(addrIn)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to allocate EIP")
	}

	log.Infof("Created EIP: %s", *addrOut.PublicIp)

	tagIn := &ec2.CreateTagsInput{
		Resources: []*string{addrOut.AllocationId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("kubernetes.io/cluster/" + input.ClusterName),
				Value: aws.String("owned"),
			},
		},
	}

	_, err = e.CreateTags(tagIn)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create tags for EIP")
	}

	log.Infof("Tagged EIP: %s", *addrOut.PublicIp)
	output := &CreateElasticIPActivityOutput{
		PublicIp:     *addrOut.PublicIp,
		AllocationId: *addrOut.AllocationId,
	}

	return output, nil
}
