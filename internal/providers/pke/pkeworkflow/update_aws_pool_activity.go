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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const UpdatePoolActivityName = "pke-update-aws-pool-activity"

type UpdatePoolActivity struct {
	clusters Clusters
}

func NewUpdatePoolActivity(clusters Clusters) *UpdatePoolActivity {
	return &UpdatePoolActivity{
		clusters: clusters,
	}
}

type UpdatePoolActivityInput struct {
	ClusterID        uint
	Pool             NodePool
	AutoScalingGroup string
}

func (a *UpdatePoolActivity) Execute(ctx context.Context, input UpdatePoolActivityInput) error {
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't get AWS client for %t", cluster))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to connect to AWS")
	}

	autoscalingSrv := autoscaling.New(client)

	_, err = autoscalingSrv.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(input.AutoScalingGroup),
		MinSize:              aws.Int64(int64(input.Pool.MinCount)),
		MaxSize:              aws.Int64(int64(input.Pool.MaxCount)),
	})
	if err != nil {
		return emperror.Wrapf(err, "setting min/max capacity of pool %q", input.Pool.Name)
	}

	_, err = autoscalingSrv.SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: aws.String(input.AutoScalingGroup),
		DesiredCapacity:      aws.Int64(int64(input.Pool.Count)),
		HonorCooldown:        aws.Bool(false),
	})
	if err != nil {
		return emperror.Wrapf(err, "setting desired capacity of pool %q", input.Pool.Name)
	}

	return nil
}
