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

package googleadapter

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// SecretStore uses the common secret store to access secrets.
type SecretStore struct {
	store common.SecretStore
}

func NewSecretStore(store common.SecretStore) SecretStore {
	return SecretStore{
		store: store,
	}
}

// GetSecret returns a secret.
func (s SecretStore) GetSecret(ctx context.Context, secretID string) (google.Secret, error) {
	values, err := s.GetRawSecret(ctx, secretID)
	if err != nil {
		return google.Secret{}, err
	}

	return google.Secret{
		Type:                   values[secrettype.Type],
		ProjectId:              values[secrettype.ProjectId],
		PrivateKeyId:           values[secrettype.PrivateKeyId],
		PrivateKey:             values[secrettype.PrivateKey],
		ClientEmail:            values[secrettype.ClientEmail],
		ClientId:               values[secrettype.ClientId],
		AuthUri:                values[secrettype.AuthUri],
		TokenUri:               values[secrettype.TokenUri],
		AuthProviderX50CertUrl: values[secrettype.AuthX509Url],
		ClientX509CertUrl:      values[secrettype.ClientX509Url],
	}, nil
}

// GetRawSecret returns the raw values of a secret.
func (s SecretStore) GetRawSecret(ctx context.Context, secretID string) (map[string]string, error) {
	return s.store.GetSecretValues(ctx, secretID)
}
