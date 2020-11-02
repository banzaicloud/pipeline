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

package eksadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

type nodePoolProcessor struct {
	db            *gorm.DB
	imageSelector eks.ImageSelector
}

// NewNodePoolProcessor returns a new cluster.NodePoolProcessor
// that processes an EKS node pool request.
//
// Note: once persistence is properly separated from Gorm,
// this should be moved to the EKS package,
// since it contains business processing rules.
func NewNodePoolProcessor(db *gorm.DB, imageSelector eks.ImageSelector) nodePoolProcessor {
	return nodePoolProcessor{
		db:            db,
		imageSelector: imageSelector,
	}
}

// ProcessNewNodePool prepares the new node pool for creation by filling in
// server side static default values.
func (p nodePoolProcessor) ProcessNewNodePool(
	ctx context.Context,
	cluster cluster.Cluster,
	nodePool eks.NewNodePool,
) (updatedNodePool eks.NewNodePool, err error) {
	var eksCluster eksmodel.EKSClusterModel

	err = p.db.
		Where(eksmodel.EKSClusterModel{ClusterID: cluster.ID}).
		Preload("Subnets").
		First(&eksCluster).Error
	if gorm.IsRecordNotFoundError(err) {
		return nodePool, errors.NewWithDetails(
			"cluster model is inconsistent",
			"clusterId", cluster.ID,
		)
	}
	if err != nil {
		return nodePool, errors.WrapWithDetails(
			err, "failed to get cluster info",
			"clusterId", cluster.ID,
			"nodePoolName", nodePool.Name,
		)
	}

	// Default node pool image
	if nodePool.Image == "" {
		criteria := eks.ImageSelectionCriteria{
			Region:            cluster.Location,
			InstanceType:      nodePool.InstanceType,
			KubernetesVersion: eksCluster.Version,
		}

		image, err := p.imageSelector.SelectImage(ctx, criteria)
		if err != nil {
			return nodePool, err
		}

		nodePool.Image = image
	}

	// Resolve subnet ID or fallback to one
	if nodePool.SubnetID == "" {
		// TODO: is this necessary?
		if len(eksCluster.Subnets) == 0 {
			return nodePool, errors.New("cannot resolve subnet")
		}

		// TODO: better algorithm for choosing a subnet?
		nodePool.SubnetID = *eksCluster.Subnets[0].SubnetId
	}

	return nodePool, nil
}
