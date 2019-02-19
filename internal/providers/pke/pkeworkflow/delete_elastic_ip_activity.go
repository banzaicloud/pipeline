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

const DeleteElasticIPActivityName = "pke-delete-eip-activity"

type DeleteElasticIPActivity struct {
	clusters Clusters
}

func NewDeleteElasticIPActivity(clusters Clusters) *DeleteElasticIPActivity {
	return &DeleteElasticIPActivity{
		clusters: clusters,
	}
}

type DeleteElasticIPActivityInput struct {
	ClusterID uint
}

func (a *DeleteElasticIPActivity) Execute(ctx context.Context, input DeleteElasticIPActivityInput) error {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't delete VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to connect to AWS")
	}

	clusterName := c.GetName()
	e := ec2.New(client)

	// look up owned EIP
	descAddrIn := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:kubernetes.io/cluster/" + clusterName),
				Values: aws.StringSlice([]string{"owned"}),
			},
		},
	}
	descAddrOut, err := e.DescribeAddresses(descAddrIn)
	if err != nil {
		return emperror.Wrap(err, "failed to query EIP based on tag")
	}

	if descAddrOut == nil || len(descAddrOut.Addresses) == 0 {
		log.Infof("no owned EIP found")
		return nil
	}

	for _, ip := range descAddrOut.Addresses {
		addrIn := &ec2.ReleaseAddressInput{
			AllocationId: ip.AllocationId,
		}
		_, err := e.ReleaseAddress(addrIn)
		if err != nil {
			return emperror.Wrapf(err, "failed to release EIP %s (%s)", ip.AllocationId, ip.PublicIp)
		}
		log.Infof("Released EIP: %s (%s)", ip.AllocationId, ip.PublicIp)
	}

	return nil
}
