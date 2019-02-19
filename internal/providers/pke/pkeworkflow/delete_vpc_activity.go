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

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const DeleteVPCActivityName = "pke-delete-vpc-activity"

type DeleteVPCActivity struct {
	clusters Clusters
}

func NewDeleteVPCActivity(clusters Clusters) *DeleteVPCActivity {
	return &DeleteVPCActivity{
		clusters: clusters,
	}
}

type DeleteVPCActivityInput struct {
	ClusterID uint
}

func (a *DeleteVPCActivity) Execute(ctx context.Context, input DeleteVPCActivityInput) error {
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

	cfClient := cloudformation.New(client)

	clusterName := c.GetName()
	stackName := "pke-vpc-" + clusterName
	stackInput := &cloudformation.DeleteStackInput{
		StackName: &stackName,
	}

	_, err = cfClient.DeleteStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		default:
			return err
		}
	}
	return nil
}
