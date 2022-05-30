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
	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
)

type nodePoolValidator struct {
	db                 *gorm.DB
	spotPriceValidator cloudinfo.SpotPriceValidator
}

// NewNodePoolValidator returns a new cluster.NodePoolValidator
// that validates an EKS node pool request.
//
// Note: once persistence is properly separated from Gorm,
// this should be moved to the EKS package,
// since it contains business validation rules.
func NewNodePoolValidator(db *gorm.DB, spotPriceValidator cloudinfo.SpotPriceValidator) nodePoolValidator {
	return nodePoolValidator{
		db:                 db,
		spotPriceValidator: spotPriceValidator,
	}
}

// ValidateNodePoolCreate validates the specified new node pool to contain the
// necessary fields and values.
func (v nodePoolValidator) ValidateNodePoolCreate(
	ctx context.Context,
	c cluster.Cluster,
	nodePool eks.NewNodePool,
) (err error) {
	validationTypeCount := 3

	errs := make([]error, 0, validationTypeCount)

	err = nodePool.Validate()
	if _, ok := err.(cluster.ValidationError); ok {
		errs = append(errs, err)
	}

	err = v.spotPriceValidator.ValidateSpotPrice(
		ctx,
		c.Cloud,
		"eks",
		c.Location,
		nodePool.InstanceType,
		c.Location,
		nodePool.SpotPrice,
	)
	if err != nil {
		errs = append(errs, err)
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
			errs = append(errs, errors.New("subnet cannot be found in the cluster"))
		}
	}

	return newNodePoolValidationErrorOrNil("invalid node pool creation request", errs...)
}

// newNodePoolValidationErrorOrNil returns a single error for the validation
// errors with the specified message and validation errors. It returns nil if no
// validation error is provided or if all given validation errors are nil.
func newNodePoolValidationErrorOrNil(message string, validationErrors ...error) error {
	violations := make([]string, 0, len(validationErrors))

	for _, validationError := range validationErrors {
		if validationError != nil {
			violations = append(violations, validationError.Error())
		}
	}

	if len(violations) != 0 {
		return cluster.NewValidationError(message, violations)
	}

	return nil
}
