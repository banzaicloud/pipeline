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

package clusterworkflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/banzaicloud/pipeline/internal/cluster"
	eksworkflow "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/providers"
)

const DeleteNodePoolActivityName = "delete-node-pool"

type DeleteNodePoolActivity struct {
	clusters          cluster.Store
	nodePools         cluster.NodePoolStore
	awsSessionFactory AWSSessionFactory
}

type AWSSessionFactory interface {
	New(organizationID uint, secretID string, region string) (*session.Session, error)
}

// NewDeleteNodePoolActivity returns a new DeleteNodePoolActivity.
func NewDeleteNodePoolActivity(
	clusters cluster.Store,
	nodePools cluster.NodePoolStore,
	awsSessionFactory AWSSessionFactory,
) DeleteNodePoolActivity {
	return DeleteNodePoolActivity{
		clusters:          clusters,
		nodePools:         nodePools,
		awsSessionFactory: awsSessionFactory,
	}
}

type DeleteNodePoolActivityInput struct {
	ClusterID    uint
	NodePoolName string
}

func (a DeleteNodePoolActivity) Execute(ctx context.Context, input DeleteNodePoolActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		input := eksworkflow.DeleteStackActivityInput{
			EKSActivityInput: eksworkflow.EKSActivityInput{
				OrganizationID:            c.OrganizationID,
				SecretID:                  c.SecretID.ResourceID,
				Region:                    c.Location,
				ClusterName:               c.Name,
				AWSClientRequestTokenBase: c.UID,
			},
			StackName: eksworkflow.GenerateNodePoolStackName(c.Name, input.NodePoolName),
		}

		err := eksworkflow.NewDeleteStackActivity(a.awsSessionFactory).Execute(ctx, input)
		if err != nil {
			return cadence.WrapClientError(err)
		}

	default:
		return cadence.WrapClientError(errors.WithStack(cluster.NotSupportedDistributionError{
			ID:           c.ID,
			Cloud:        c.Cloud,
			Distribution: c.Distribution,

			Message: "the node pool API does not support this distribution yet",
		}))
	}

	err = a.nodePools.DeleteNodePool(ctx, input.ClusterID, input.NodePoolName)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	return nil
}
