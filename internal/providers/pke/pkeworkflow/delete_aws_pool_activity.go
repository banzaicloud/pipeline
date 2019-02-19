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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const DeletePoolActivityName = "pke-delete-aws-pool-activity"

type DeletePoolActivity struct {
	clusters Clusters
}

func NewDeletePoolActivity(clusters Clusters) *DeletePoolActivity {
	return &DeletePoolActivity{
		clusters: clusters,
	}
}

type DeletePoolActivityInput struct {
	ClusterID uint
	Pool      NodePool
}

func (a *DeletePoolActivity) Execute(ctx context.Context, input DeletePoolActivityInput) error {
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	stackName := fmt.Sprintf("pke-pool-%s-worker-%s", cluster.GetName(), input.Pool.Name)
	if input.Pool.Master {
		stackName = fmt.Sprintf("pke-master-%s", cluster.GetName())
	}

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't get AWS client for %t", cluster))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	stackInput := &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
		//ClientRequestToken: aws.String(string(activity.GetInfo(ctx).ActivityID)),
	}

	_, err = cfClient.DeleteStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		default:
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
