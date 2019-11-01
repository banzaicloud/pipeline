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

package validation

import (
	"emperror.dev/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

// ValidateSubdomain verifies if the provided subdomain string complies with DNS-1123
func ValidateSubdomain(subdomain string) error {
	violations := validation.IsDNS1123Subdomain(subdomain)

	if len(violations) > 0 {
		errs := make([]error, 0, len(violations))

		for _, violation := range violations {
			errs = append(errs, errors.NewWithDetails(violation, "subdomain", subdomain))
		}

		return errors.Combine(errs...)
	}

	return nil
}
