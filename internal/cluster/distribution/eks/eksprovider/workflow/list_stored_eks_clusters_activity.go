// Copyright © 2020 Banzai Cloud
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

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

// ListStoredEKSClustersActivityName is the name of the activity which lists the
// stored EKS clusters.
const ListStoredEKSClustersActivityName = "eks-list-stored-eks-clusters"

// ListStoredEKSClustersActivity collects the static high level component objects
// required for retrieveing a stored EKS cluster.
type ListStoredEKSClustersActivity struct {
	db *gorm.DB
}

// NewListStoredEKSClustersActivity instantiates a stored EKS cluster retrieving
// activity.
func NewListStoredEKSClustersActivity(db *gorm.DB) (activity *ListStoredEKSClustersActivity) {
	return &ListStoredEKSClustersActivity{
		db: db,
	}
}

// ListStoredEKSClustersActivityInput collects the required parameters for setting
// a node pool\s status.
type ListStoredEKSClustersActivityInput struct {
	OptionalListedGenericClusterIDs []uint
}

// ListStoredEKSClustersActivityOutput collects the returned output of the
// EKS cluster retrieval activity.
type ListStoredEKSClustersActivityOutput struct {
	// EKSClusters are a map of unique **generic cluster ID** keys and EKS
	// cluster model values.
	//
	// The reason for not keying them with their primary key EKS cluster IDs is
	// usability, EKS cluster ID is an internal detail and most of the
	// interfaces/operations work with generic cluster ID thus that is much
	// easier/straightforward to use as long as each EKS cluster has a unique
	// generic cluster ID which is the case currently.
	EKSClusters map[uint]eksmodel.EKSClusterModel
}

// Execute executes the activity.
func (a ListStoredEKSClustersActivity) Execute(
	ctx context.Context, input ListStoredEKSClustersActivityInput,
) (output *ListStoredEKSClustersActivityOutput, err error) {
	eksClusterDB := a.db.
		Preload("Cluster").
		Preload("NodePools").
		Preload("Subnets")

	if len(input.OptionalListedGenericClusterIDs) != 0 { // Note: 0 == list every cluster.
		eksClusterDB = eksClusterDB.Where("cluster_id in (?)", input.OptionalListedGenericClusterIDs)
	}

	var eksClusters []eksmodel.EKSClusterModel
	err = eksClusterDB.Find(&eksClusters).Error
	if err != nil {
		return nil, errors.Wrap(err, "listing stored eks clusters failed")
	}

	output = &ListStoredEKSClustersActivityOutput{
		EKSClusters: make(map[uint]eksmodel.EKSClusterModel, len(eksClusters)),
	}
	for _, eksCluster := range eksClusters {
		output.EKSClusters[eksCluster.ClusterID] = eksCluster
	}

	return output, nil
}

// Register registers the activity.
func (a ListStoredEKSClustersActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: ListStoredEKSClustersActivityName})
}

// listStoredEKSClusters lists the stored EKS clusters optionally filtered for
// the specified generic cluster IDs.
//
// For more information on the returned map and its keying see the corresponding
// activity output type.
//
// This is a convenience wrapper around the corresponding activity.
func listStoredEKSClusters(
	ctx workflow.Context,
	optionalListedGenericClusterIDs ...uint,
) (map[uint]eksmodel.EKSClusterModel, error) {
	var activityOutput ListStoredEKSClustersActivityOutput
	err := listStoredEKSClustersAsync(ctx, optionalListedGenericClusterIDs...).Get(ctx, &activityOutput)
	if err != nil {
		return nil, err
	}

	return activityOutput.EKSClusters, nil
}

// listStoredEKSClustersAsync returns a future object for listing the stored EKS
// clusters optionally filtered for the specified generic cluster IDs.
//
// This is a convenience wrapper around the corresponding activity.
func listStoredEKSClustersAsync(ctx workflow.Context, optionalListedGenericClusterIDs ...uint) workflow.Future {
	return workflow.ExecuteActivity(ctx, ListStoredEKSClustersActivityName, ListStoredEKSClustersActivityInput{
		OptionalListedGenericClusterIDs: optionalListedGenericClusterIDs,
	})
}
