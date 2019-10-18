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

package secrettypedriver

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/secrettype"
)

func TestEndpoints_ListSecretTypes(t *testing.T) {
	ctx := context.Background()

	expectedSecretTypes := map[string]secrettype.TypeDefinition{
		"secretType": {
			Fields: []secrettype.TypeField{
				{
					Name:        "field",
					Required:    true,
					Opaque:      true,
					Description: "Field description",
				},
			},
		},
	}

	service := new(secrettype.MockTypeService)
	service.On("ListSecretTypes", ctx).Return(expectedSecretTypes, nil)

	e := MakeEndpoints(service).ListSecretTypes

	resp, err := e(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, expectedSecretTypes, resp)

	service.AssertExpectations(t)
}

func TestEndpoints_GetSecretType(t *testing.T) {
	ctx := context.Background()
	secretType := "secretType"

	expectedSecretType := secrettype.TypeDefinition{
		Fields: []secrettype.TypeField{
			{
				Name:        "field",
				Required:    true,
				Opaque:      true,
				Description: "Field description",
			},
		},
	}

	service := new(secrettype.MockTypeService)
	service.On("GetSecretType", ctx, secretType).Return(expectedSecretType, nil)

	e := MakeEndpoints(service).GetSecretType

	resp, err := e(context.Background(), getSecretTypeRequest{secretType})
	require.NoError(t, err)

	assert.Equal(t, expectedSecretType, resp)

	service.AssertExpectations(t)
}

func TestEndpoints_GetSecretType_NotFound(t *testing.T) {
	ctx := context.Background()
	secretType := "secretType"

	service := new(secrettype.MockTypeService)
	service.On("GetSecretType", ctx, secretType).Return(secrettype.TypeDefinition{}, secrettype.ErrNotSupportedSecretType)

	e := MakeEndpoints(service).GetSecretType

	resp, err := e(context.Background(), getSecretTypeRequest{secretType})
	require.NoError(t, err)

	assert.True(t, errors.Is(resp.(endpoint.Failer).Failed(), secrettype.ErrNotSupportedSecretType))

	service.AssertExpectations(t)
}
