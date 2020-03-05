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
	"github.com/banzaicloud/pipeline/internal/providers/amazon/amazonadapter"
)

// nodePoolValidator validates a node pool request according to its own distribution.
type nodePoolValidator struct {
	db *gorm.DB
}

// NewNodePoolValidator returns a new cluster.NodePoolValidator.
func NewNodePoolValidator(db *gorm.DB) cluster.NodePoolValidator {
	return nodePoolValidator{
		db: db,
	}
}

// ValidateNew validates a new node pool descriptor.
func (v nodePoolValidator) ValidateNew(
	_ context.Context,
	c cluster.Cluster,
	rawNodePool cluster.NewRawNodePool,
) error {
	var nodePool eks.NewNodePool

	err := mapstructure.Decode(rawNodePool, &nodePool)
	if err != nil {
		return errors.Wrap(err, "failed to decode node pool")
	}

	message := "invalid node pool creation request"
	var violations []string

	verr := nodePool.Validate()
	if err, ok := verr.(cluster.ValidationError); ok {
		message = err.Error()
		violations = err.Violations()
	}

	var eksCluster amazonadapter.EKSClusterModel

	err = v.db.
		Where(amazonadapter.EKSClusterModel{ClusterID: c.ID}).
		Preload("Subnets").
		First(&eksCluster).Error
	if gorm.IsRecordNotFoundError(err) {
		return errors.NewWithDetails(
			"cluster model is inconsistent",
			"clusterId", c.ID,
		)
	}
	if err != nil {
		return errors.WrapWithDetails(
			err, "failed to get cluster info",
			"clusterId", c.ID,
			"nodePoolName", nodePool.Name,
		)
	}

	hasSubnet := false
	validSubnet := false

	if nodePool.Subnet.SubnetId != "" {
		hasSubnet = true

		for _, s := range eksCluster.Subnets {
			if s.SubnetId != nil && *s.SubnetId == nodePool.Subnet.SubnetId {
				validSubnet = true

				break
			}
		}
	} else if nodePool.Subnet.Cidr != "" && nodePool.Subnet.AvailabilityZone != "" {
		hasSubnet = true

		for _, s := range eksCluster.Subnets {
			if s.Cidr != nil && *s.Cidr == nodePool.Subnet.Cidr && s.AvailabilityZone != nil && *s.AvailabilityZone == nodePool.Subnet.AvailabilityZone {
				validSubnet = true

				break
			}
		}
	} else if nodePool.Subnet.Cidr != "" || nodePool.Subnet.AvailabilityZone != "" {
		violations = append(violations, "cidr and availability zone must be specified together")
	}

	if hasSubnet && !validSubnet {
		violations = append(violations, "subnet cannot be found in the cluster")
	}

	if len(violations) > 0 {
		return cluster.NewValidationError(message, violations)
	}

	return nil
}
