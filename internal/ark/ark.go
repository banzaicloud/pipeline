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

package ark

import (
	"github.com/pkg/errors"

	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/secret"
)

// IsProviderSupported checks whether the given provider is supported
func IsProviderSupported(provider string) error {

	switch provider {
	case providers.Google:
		return nil
	case providers.Amazon:
		return nil
	case providers.Azure:
		return nil
	default:
		return pkgErrors.ErrorNotSupportedCloudType
	}
}

// GetSecretWithValidation gives back a secret response with validation
func GetSecretWithValidation(secretID string, orgID uint, provider string) (*secret.SecretItemResponse, error) {

	secret, err := secret.Store.Get(orgID, secretID)
	if err != nil {
		return nil, errors.Wrap(err, "error validating create bucket request")
	}

	err = secret.ValidateSecretType(provider)
	if err != nil {
		return nil, errors.Wrap(err, "error validating create bucket request")
	}

	return secret, nil
}
