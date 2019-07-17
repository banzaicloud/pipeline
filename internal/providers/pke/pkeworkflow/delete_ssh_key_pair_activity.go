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

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
)

const DeleteSSHKeyPairActivityName = "pke-delete-ssh-key-pair-activity"

type DeleteSSHKeyPairActivity struct {
	clusters Clusters
}

func NewDeleteSSHKeyPairActivity(clusters Clusters) *DeleteSSHKeyPairActivity {
	return &DeleteSSHKeyPairActivity{
		clusters: clusters,
	}
}

type DeleteSSHKeyPairActivityInput struct {
	ClusterID uint
}

func (a *DeleteSSHKeyPairActivity) Execute(ctx context.Context, input DeleteSSHKeyPairActivityInput) error {
	// log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't create VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to connect to AWS")
	}

	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	clusterName := cluster.GetName()
	keyName := "pke-ssh-" + clusterName

	e := ec2.New(client)

	describeKeyPairsInput := &ec2.DescribeKeyPairsInput{
		KeyNames: aws.StringSlice([]string{keyName}),
	}

	describeKeyPairsOutput, err := e.DescribeKeyPairs(describeKeyPairsInput)
	if err != nil {
		if a, ok := err.(awserr.Error); ok {
			if a.Code() == "InvalidKeyPair.NotFound" {
				return nil
			}
		}
		return err
	}

	if len(describeKeyPairsOutput.KeyPairs) <= 0 {
		// somebody already deleted the key pair
		return nil
	}

	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: &keyName,
	}

	_, err = e.DeleteKeyPair(deleteKeyPairInput)
	if err != nil {
		return emperror.Wrap(err, "failed to delete key pair on AWS EC2")
	}

	return nil
}
