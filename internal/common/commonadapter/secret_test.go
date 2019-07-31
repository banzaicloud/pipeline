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

package commonadapter

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/secret"
)

//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name OrganizationalSecretStore -inpkg -testonly"

func TestSecretStore_GetSecretValues(t *testing.T) {
	organizationID := uint(1)
	secretID := "id"
	secretResponse := &secret.SecretItemResponse{
		Values: map[string]string{
			"key": "value",
		},
	}

	orgStore := &MockOrganizationalSecretStore{}
	orgStore.On("Get", organizationID, secretID).Return(secretResponse, nil)

	const orgIdKey = "orgIdKey"

	store := NewSecretStore(
		orgStore,
		OrgIDContextExtractorFunc(func(ctx context.Context) (uint, bool) {
			id, ok := ctx.Value(orgIdKey).(uint)

			return id, ok
		}),
	)

	ctx := context.WithValue(context.Background(), orgIdKey, organizationID)

	values, err := store.GetSecretValues(ctx, secretID)
	require.NoError(t, err)

	assert.Equal(t, secretResponse.Values, values)
}

func TestSecretStore_GetSecretValues_SecretNotFound(t *testing.T) {
	organizationID := uint(1)
	secretID := "id"

	orgStore := &MockOrganizationalSecretStore{}
	orgStore.On("Get", organizationID, secretID).Return(nil, secret.ErrSecretNotExists)

	const orgIdKey = "orgIdKey"

	store := NewSecretStore(
		orgStore,
		OrgIDContextExtractorFunc(func(ctx context.Context) (uint, bool) {
			id, ok := ctx.Value(orgIdKey).(uint)

			return id, ok
		}),
	)

	ctx := context.WithValue(context.Background(), orgIdKey, organizationID)

	values, err := store.GetSecretValues(ctx, secretID)
	require.Error(t, err)

	assert.Nil(t, values)
	assert.True(t, errors.As(err, &common.SecretNotFoundError{}))
	assert.Equal(
		t,
		[]interface{}{"secretId", secretID, "organizationId", organizationID},
		errors.GetDetails(err),
	)
}

func TestSecretStore_GetSecretValues_SomethingWentWrong(t *testing.T) {
	organizationID := uint(1)
	secretID := "id"

	origErr := errors.NewPlain("something went wrong")

	orgStore := &MockOrganizationalSecretStore{}
	orgStore.On("Get", organizationID, secretID).Return(nil, origErr)

	const orgIdKey = "orgIdKey"

	store := NewSecretStore(
		orgStore,
		OrgIDContextExtractorFunc(func(ctx context.Context) (uint, bool) {
			id, ok := ctx.Value(orgIdKey).(uint)

			return id, ok
		}),
	)

	ctx := context.WithValue(context.Background(), orgIdKey, organizationID)

	values, err := store.GetSecretValues(ctx, secretID)
	require.Error(t, err)

	assert.Nil(t, values)
	assert.Equal(t, origErr, errors.Cause(err))
	assert.Equal(
		t,
		[]interface{}{"organizationId", organizationID, "secretId", secretID},
		errors.GetDetails(err),
	)
}
