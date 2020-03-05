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

package cluster

import (
	"context"

	"emperror.dev/errors"
)

// NodePoolValidators combines different node pool validators into one.
type NodePoolValidators []NodePoolValidator

// ValidateNew validates a new node pool descriptor.
func (v NodePoolValidators) ValidateNew(ctx context.Context, cluster Cluster, rawNodePool NewRawNodePool) error {
	var violations []string

	for _, validator := range v {
		err := validator.ValidateNew(ctx, cluster, rawNodePool)
		if err != nil {
			violations = append(violations, unwrapViolations(err)...)
		}
	}

	if len(violations) > 0 {
		return errors.WithStack(ValidationError{
			message:    "invalid node pool",
			violations: violations,
		})
	}

	return nil
}

type commonNodePoolValidator struct {
	labelValidator LabelValidator
}

// NewCommonNodePoolValidator returns a new NodePoolValidator
// that validates common fields found in all node pool types.
func NewCommonNodePoolValidator(labelValidator LabelValidator) NodePoolValidator {
	return commonNodePoolValidator{
		labelValidator: labelValidator,
	}
}

// +testify:mock:testOnly=true

// LabelValidator validates Kubernetes object labels.
type LabelValidator interface {
	// ValidateKey validates a label key.
	ValidateKey(key string) error

	// ValidateValue validates a label value.
	ValidateValue(value string) error
}

func (v commonNodePoolValidator) ValidateNew(_ context.Context, _ Cluster, rawNodePool NewRawNodePool) error {
	var violations []string

	if rawNodePool.GetName() == "" {
		violations = append(violations, "name must be a non-empty string")
	}

	for key, value := range rawNodePool.GetLabels() {
		if err := v.labelValidator.ValidateKey(key); err != nil {
			violations = append(violations, unwrapViolations(err)...)
		}

		if err := v.labelValidator.ValidateValue(value); err != nil {
			violations = append(violations, unwrapViolations(err)...)
		}
	}

	if len(violations) > 0 {
		return errors.WithStack(ValidationError{
			message:    "invalid node pool",
			violations: violations,
		})
	}

	return nil
}

type distributionNodePoolValidator struct {
	validators map[string]NodePoolValidator
}

// NewDistributionNodePoolValidator returns a new NodePoolValidator
// that allows registering validators for Kubernetes distributions.
func NewDistributionNodePoolValidator(validators map[string]NodePoolValidator) NodePoolValidator {
	return distributionNodePoolValidator{
		validators: validators,
	}
}

func (v distributionNodePoolValidator) ValidateNew(ctx context.Context, cluster Cluster, rawNodePool NewRawNodePool) error {
	validator, ok := v.validators[cluster.Distribution]
	if !ok {
		return errors.WithStack(NotSupportedDistributionError{
			ID:           cluster.ID,
			Cloud:        cluster.Cloud,
			Distribution: cluster.Distribution,

			Message: "cannot validate unsupported distribution",
		})
	}

	return validator.ValidateNew(ctx, cluster, rawNodePool)
}

// unwrapViolations is a helper func to unwrap violations from a validation error
func unwrapViolations(err error) []string {
	var verr interface {
		Violations() []string
	}

	if errors.As(err, &verr) {
		return verr.Violations()
	}

	return []string{err.Error()}
}
