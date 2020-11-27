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

package dns

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

func TestSecretMapperSucceed(t *testing.T) {
	secretStore := SecretStore{
		IDByName: map[string]string{
			"fake-secret-name": "fake-secret-id",
		},
	}
	mapper := NewSecretMapper(secretStore)
	spec, err := mapper.MapSpec(context.TODO(), integratedservices.IntegratedServiceSpec{
		"externalDns": map[string]interface{}{
			"provider": map[string]string{
				"secretId": "fake-secret-name",
			},
		},
	})
	require.NoError(t, err)

	boundSpec, err := bindIntegratedServiceSpec(spec)
	require.NoError(t, err)

	require.Equal(t, "fake-secret-id", boundSpec.ExternalDNS.Provider.SecretID)
}

func TestSecretMapperFail(t *testing.T) {
	secretStore := SecretStore{
		IDByName: map[string]string{
			"fake-secret-name": "fake-secret-id",
		},
	}
	mapper := NewSecretMapper(secretStore)
	_, err := mapper.MapSpec(context.TODO(), integratedservices.IntegratedServiceSpec{
		"externalDns": map[string]interface{}{
			"provider": map[string]string{
				"secretId": "unknown-secret-name",
			},
		},
	})
	require.Error(t, err)
}

type SecretStore struct {
	IDByName map[string]string
}

func (s SecretStore) GetIDByName(ctx context.Context, secretName string) (string, error) {
	if id, ok := s.IDByName[secretName]; ok {
		return id, nil
	}
	return "", errors.New("notfound")
}

func (s SecretStore) GetSecretValues(ctx context.Context, secretID string) (map[string]string, error) {
	panic("implement me")
}

func (s SecretStore) GetNameByID(ctx context.Context, secretID string) (string, error) {
	panic("implement me")
}

func (s SecretStore) Delete(ctx context.Context, secretID string) error {
	panic("implement me")
}
