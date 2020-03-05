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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/cluster"
	eks2 "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/providers/amazon/amazonadapter"
	"github.com/banzaicloud/pipeline/pkg/providers"
)

// DistributionNodePoolProcessor processes a node pool request according to its own distribution.
type DistributionNodePoolProcessor struct {
	db *gorm.DB
}

// NewDistributionNodePoolProcessor returns a new DistributionNodePoolProcessor.
func NewDistributionNodePoolProcessor(db *gorm.DB) DistributionNodePoolProcessor {
	return DistributionNodePoolProcessor{
		db: db,
	}
}

// ProcessNew processes a new node pool descriptor.
func (v DistributionNodePoolProcessor) ProcessNew(
	_ context.Context,
	c cluster.Cluster,
	rawNodePool cluster.NewRawNodePool,
) (cluster.NewRawNodePool, error) {
	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var nodePool eks2.NewNodePool

		err := mapstructure.Decode(rawNodePool, &nodePool)
		if err != nil {
			return rawNodePool, errors.Wrap(err, "failed to decode node pool")
		}

		var eksCluster amazonadapter.EKSClusterModel

		err = v.db.
			Where(amazonadapter.EKSClusterModel{ClusterID: c.ID}).
			Preload("Subnets").
			First(&eksCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return rawNodePool, errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", c.ID,
			)
		}
		if err != nil {
			return rawNodePool, errors.WrapWithDetails(
				err, "failed to get cluster info",
				"clusterId", c.ID,
				"nodePoolName", nodePool.Name,
			)
		}

		// Default node pool image
		if nodePool.Image == "" {
			image, err := eks2.GetDefaultImageID(c.Location, eksCluster.Version)
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

	return rawNodePool, errors.WithStack(cluster.NotSupportedDistributionError{
		ID:           c.ID,
		Cloud:        c.Cloud,
		Distribution: c.Distribution,

		Message: "cannot process unsupported distribution",
	})
}
