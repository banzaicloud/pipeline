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
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

type nodePoolProcessor struct {
	db *gorm.DB
}

// NewNodePoolProcessor returns a new cluster.NodePoolProcessor
// that processes an EKS node pool request.
//
// Note: once persistence is properly separated from Gorm,
// this should be moved to the EKS package,
// since it contains business processing rules.
func NewNodePoolProcessor(db *gorm.DB) cluster.NodePoolProcessor {
	return nodePoolProcessor{
		db: db,
	}
}

func (p nodePoolProcessor) ProcessNew(
	_ context.Context,
	cluster cluster.Cluster,
	rawNodePool cluster.NewRawNodePool,
) (cluster.NewRawNodePool, error) {
	var nodePool eks.NewNodePool

	err := mapstructure.Decode(rawNodePool, &nodePool)
	if err != nil {
		return rawNodePool, errors.Wrap(err, "failed to decode node pool")
	}

	var eksCluster eksmodel.EKSClusterModel

	err = p.db.
		Where(eksmodel.EKSClusterModel{ClusterID: cluster.ID}).
		Preload("Subnets").
		First(&eksCluster).Error
	if gorm.IsRecordNotFoundError(err) {
		return rawNodePool, errors.NewWithDetails(
			"cluster model is inconsistent",
			"clusterId", cluster.ID,
		)
	}
	if err != nil {
		return rawNodePool, errors.WrapWithDetails(
			err, "failed to get cluster info",
			"clusterId", cluster.ID,
			"nodePoolName", nodePool.Name,
		)
	}

	// Default node pool image
	if nodePool.Image == "" {
		image, err := eks.GetDefaultImageID(cluster.Location, eksCluster.Version)
		if err != nil {
			return rawNodePool, err
		}

		rawNodePool["image"] = image
	}

	// Resolve subnet ID or fallback to one
	if nodePool.Subnet.SubnetId == "" && nodePool.Subnet.Cidr != "" && nodePool.Subnet.AvailabilityZone != "" {
		for _, s := range eksCluster.Subnets {
			if s.Cidr != nil && *s.Cidr == nodePool.Subnet.Cidr && s.AvailabilityZone != nil && *s.AvailabilityZone == nodePool.Subnet.AvailabilityZone {
				rawNodePool["subnet"] = map[string]interface{}{
					"subnetId":         *s.SubnetId,
					"cidr":             nodePool.Subnet.Cidr,
					"availabilityZone": nodePool.Subnet.AvailabilityZone,
				}
			}
		}
	} else if nodePool.Subnet.SubnetId == "" {
		// TODO: is this necessary?
		if len(eksCluster.Subnets) == 0 {
			return rawNodePool, errors.New("cannot resolve subnet")
		}

		rawNodePool["subnet"] = map[string]interface{}{
			"subnetId":         *eksCluster.Subnets[0].SubnetId,
			"cidr":             *eksCluster.Subnets[0].Cidr,
			"availabilityZone": *eksCluster.Subnets[0].AvailabilityZone,
		}
	}

	return rawNodePool, nil
}
