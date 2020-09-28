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

package workflow

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

const SaveNodePoolsActivityName = "eks-save-node-pools"

type SaveNodePoolsActivity struct {
	manager Clusters
}

func NewSaveNodePoolsActivity(manager Clusters) SaveNodePoolsActivity {
	return SaveNodePoolsActivity{
		manager: manager,
	}
}

type SaveNodePoolsActivityInput struct {
	ClusterID uint

	NodePoolsToDelete map[string]AutoscaleGroup
	NodePoolsToUpdate map[string]AutoscaleGroup
	NodePoolsToCreate map[string]AutoscaleGroup
	NodePoolsToKeep   map[string]bool
}

func (a SaveNodePoolsActivity) Execute(ctx context.Context, input SaveNodePoolsActivityInput) error {
	cluster, err := a.manager.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	if eksCluster, ok := cluster.(interface {
		GetModel() *eksmodel.EKSClusterModel
	}); ok {
		modelCluster := eksCluster.GetModel()
		nodePoolCount := len(input.NodePoolsToCreate) + len(input.NodePoolsToDelete) + len(input.NodePoolsToKeep) +
			len(input.NodePoolsToUpdate)
		updatedNodepools := make([]*eksmodel.AmazonNodePoolsModel, 0, nodePoolCount)

		for _, np := range modelCluster.NodePools {
			if input.NodePoolsToKeep[np.Name] {
				updatedNodepools = append(updatedNodepools, np)

				continue
			}

			_, ok := input.NodePoolsToDelete[np.Name]
			if ok {
				np.Delete = true
				updatedNodepools = append(updatedNodepools, np)
				continue
			}
			asg, ok := input.NodePoolsToUpdate[np.Name]
			if ok {
				np.Autoscaling = asg.Autoscaling
				np.NodeMinCount = asg.NodeMinCount
				np.NodeMaxCount = asg.NodeMaxCount
				np.Count = asg.Count
				updatedNodepools = append(updatedNodepools, np)
			}
		}

		for _, asg := range input.NodePoolsToCreate {
			np := &eksmodel.AmazonNodePoolsModel{
				CreatedBy:        asg.CreatedBy,
				Name:             asg.Name,
				StackID:          "",
				NodeInstanceType: asg.NodeInstanceType,
				NodeImage:        asg.NodeImage,
				NodeSpotPrice:    asg.NodeSpotPrice,
				Autoscaling:      asg.Autoscaling,
				NodeMinCount:     asg.NodeMinCount,
				NodeMaxCount:     asg.NodeMaxCount,
				Count:            asg.Count,
				Status:           eks.NodePoolStatusCreating,
				StatusMessage:    "",
				// NodeVolumeSize:   asg.NodeVolumeSize, // Note: not stored in DB.
				// Labels:           asg.Labels, // Note: not stored in DB.
				Delete: false,
			}
			updatedNodepools = append(updatedNodepools, np)
		}
		modelCluster.NodePools = updatedNodepools
	}

	return cluster.Persist()
}
