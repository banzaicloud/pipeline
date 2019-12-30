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

// CommonNodePoolValidator validates fields found in all node pool types.
type CommonNodePoolValidator struct {
	labelValidator LabelValidator
}

// NewCommonNodePoolValidator returns a new CommonNodePoolValidator.
func NewCommonNodePoolValidator(labelValidator LabelValidator) CommonNodePoolValidator {
	return CommonNodePoolValidator{
		labelValidator: labelValidator,
	}
}

// LabelValidator validates Kubernetes object labels.
type LabelValidator interface {
	// ValidateKey validates a label key.
	ValidateKey(key string) error

	// ValidateValue validates a label value.
	ValidateValue(value string) error
}

func (v CommonNodePoolValidator) Validate(_ context.Context, _ Cluster, rawNodePool NewRawNodePool) error {
	var violations []string

	if name, ok := rawNodePool["name"].(string); !ok || name == "" {
		violations = append(violations, "name must be a non-empty string")
	}

	// Helper func to unwrap violations from a validation error
	unwrapViolations := func(err error) []string {
		type validationError interface {
			Violations() []string
		}

		if verr, ok := err.(validationError); ok {
			return verr.Violations()
		}

		return []string{err.Error()}
	}

	if labels, ok := rawNodePool["labels"].(map[string]string); ok {
		for key, value := range labels {
			if err := v.labelValidator.ValidateKey(key); err != nil {
				violations = append(violations, unwrapViolations(err)...)
			}

			if err := v.labelValidator.ValidateValue(value); err != nil {
				violations = append(violations, unwrapViolations(err)...)
			}
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
