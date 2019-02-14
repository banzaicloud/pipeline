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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence/activity"
)

const CreateElasticIPActivityName = "pke-create-eip-activity"

type CreateElasticIPActivity struct {
	clusters Clusters
}

func NewCreateElasticIPActivity(clusters Clusters) *CreateVPCActivity {
	return &CreateVPCActivity{
		clusters: clusters,
	}
}

type CreateElasticIPActivityInput struct {
	ClusterID uint
}

func (a *CreateElasticIPActivity) Execute(ctx context.Context, input CreateElasticIPActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return "", errors.New(fmt.Sprintf("can't create VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", emperror.Wrap(err, "failed to connect to AWS")
	}

	clusterName := c.GetName()
	e := ec2.New(client)

	// check EIP is already allocated or not
	descAddrIn := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: aws.StringSlice([]string{"kubernetes.io/cluster/" + clusterName}),
			},
		},
	}
	descAddrOut, err := e.DescribeAddresses(descAddrIn)
	if err != nil {
		return "", emperror.Wrap(err, "failed to query EIP based on tag")
	}

	if descAddrOut != nil && len(descAddrOut.Addresses) > 0 {
		log.Infof("EIP already exists: %s", *descAddrOut.Addresses[0].PublicIp)
		return *descAddrOut.Addresses[0].PublicIp, nil
	}

	addrIn := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	}
	addrOut, err := e.AllocateAddress(addrIn)
	if err != nil {
		return "", emperror.Wrap(err, "failed to allocate EIP")
	}
	log.Infof("Created EIP: %s", *addrOut.PublicIp)

	tagIn := &ec2.CreateTagsInput{
		Resources: []*string{addrOut.AllocationId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("kubernetes.io/cluster/" + clusterName),
				Value: aws.String("owned"),
			},
		},
	}
	_, err = e.CreateTags(tagIn)
	if err != nil {
		return "", emperror.Wrap(err, "failed to create tags for EIP")
	}
	log.Infof("Tagged EIP: %s", *addrOut.PublicIp)

	return *addrOut.PublicIp, nil
}
