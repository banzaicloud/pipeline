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

package kubernetes

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

// LabelValidator validates Kubernetes object labels.
type LabelValidator struct {
	ForbiddenDomains   []string
	ForbiddenLabelKeys []string
}

// ValidateKey validates a label key.
func (v LabelValidator) ValidateKey(key string) error {
	var violations []string

	for _, v := range validation.IsQualifiedName(key) {
		violations = append(violations, fmt.Sprintf("invalid label key %q: %s", key, v))
	}

	for _, domain := range v.ForbiddenDomains {
		domain = strings.ToLower(domain)

		var keyDomain string

		keyDomainParts := strings.Split(key, "/")
		if len(keyDomainParts) > 1 {
			keyDomain = strings.ToLower(keyDomainParts[0])
		}

		if keyDomain == domain || strings.HasSuffix(keyDomain, "."+domain) {
			violations = append(violations, fmt.Sprintf("forbidden label key domain in %q: %q domain is not allowed", key, domain))
		}
	}

	for _, labelKey := range v.ForbiddenLabelKeys {
		if key == labelKey {
			violations = append(violations, fmt.Sprintf("label key %q is not allowed", key))
		}
	}

	if len(violations) > 0 {
		return errors.WithStack(LabelValidationError{
			violations: violations,
		})
	}

	return nil
}

// ValidateValue validates a label value.
func (v LabelValidator) ValidateValue(value string) error {
	var violations []string

	for _, v := range validation.IsValidLabelValue(value) {
		violations = append(violations, fmt.Sprintf("invalid label value %q: %s", value, v))
	}

	if len(violations) > 0 {
		return errors.WithStack(LabelValidationError{
			violations: violations,
		})
	}

	return nil
}

// LabelValidationError is returned (with a set of underlying violations) when a label is invalid.
type LabelValidationError struct {
	violations []string
}

// Error implements the error interface.
func (e LabelValidationError) Error() string {
	return "invalid label"
}

// Violations returns details of the failed validation.
func (e LabelValidationError) Violations() []string {
	return e.violations[:]
}

// Validation tells a client that this error is related to a semantic validation of the request.
// Can be used to translate the error to status codes for example.
func (LabelValidationError) Validation() bool {
	return true
}

// ClientError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (LabelValidationError) ClientError() bool {
	return true
}
