// Copyright © 2018 Banzai Cloud
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

package providers

import (
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
)

type secretStore interface {
	Get(organizationID pkgAuth.OrganizationID, secretID pkgSecret.SecretID) (*secret.SecretItemResponse, error)
}

type secretValidator struct {
	secrets secretStore
}

// NewSecretValidator returns a struct which validates that a secret belongs to a cloud provider.
func NewSecretValidator(secrets secretStore) *secretValidator {
	return &secretValidator{secrets}
}

// ValidateSecretType validates that a secret belongs to a cloud provider.
func (v *secretValidator) ValidateSecretType(organizationID pkgAuth.OrganizationID, secretID pkgSecret.SecretID, provider string) error {
	s, err := v.secrets.Get(organizationID, secretID)
	if err == secret.ErrSecretNotExists {
		return errors.Wrap(err, "error during secret validation")
	} else if err != nil {
		return errors.WithMessage(err, "error during secret validation")
	}

	return s.ValidateSecretType(provider)
}
