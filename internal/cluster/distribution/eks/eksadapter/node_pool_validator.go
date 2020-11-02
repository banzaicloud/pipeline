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

type nodePoolValidator struct {
	db *gorm.DB
}

// NewNodePoolValidator returns a new cluster.NodePoolValidator
// that validates an EKS node pool request.
//
// Note: once persistence is properly separated from Gorm,
// this should be moved to the EKS package,
// since it contains business validation rules.
func NewNodePoolValidator(db *gorm.DB) nodePoolValidator {
	return nodePoolValidator{
		db: db,
	}
}

// ValidateNewNodePool validates the specified new node pool to contain the
// necessary fields and values.
func (v nodePoolValidator) ValidateNewNodePool(
	_ context.Context,
	c cluster.Cluster,
	nodePool eks.NewNodePool,
) (err error) {
	message := "invalid node pool creation request"
	var violations []string

	verr := nodePool.Validate()
	if err, ok := verr.(cluster.ValidationError); ok {
		message = err.Error()
		violations = err.Violations()
	}

	var eksCluster eksmodel.EKSClusterModel

	err = v.db.
		Where(eksmodel.EKSClusterModel{ClusterID: c.ID}).
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

	if nodePool.SubnetID != "" {
		validSubnet := false

		for _, s := range eksCluster.Subnets {
			if s.SubnetId != nil && *s.SubnetId == nodePool.SubnetID {
				validSubnet = true

				break
			}
		}

		if !validSubnet {
			violations = append(violations, "subnet cannot be found in the cluster")
		}
	}

	if len(violations) > 0 {
		return cluster.NewValidationError(message, violations)
	}

	return nil
}
