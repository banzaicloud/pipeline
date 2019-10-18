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

package secrettype

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeService_ListSecretTypes(t *testing.T) {
	types := map[string]TypeDefinition{
		"secretType": {
			Fields: []TypeField{
				{
					Name:        "field",
					Required:    true,
					Description: "Field description",
				},
			},
		},
	}

	var service = typeService{
		types: types,
	}

	typeDefs, err := service.ListSecretTypes(context.Background())
	require.NoError(t, err)

	assert.Equal(t, types, typeDefs)
}

func TestTypeService_GetSecretType(t *testing.T) {
	typeDefinition := TypeDefinition{
		Fields: []TypeField{
			{
				Name:        "field",
				Required:    true,
				Description: "Field description",
			},
		},
	}

	var service = typeService{
		types: map[string]TypeDefinition{
			"secretType": typeDefinition,
		},
	}

	typeDef, err := service.GetSecretType(context.Background(), "secretType")
	require.NoError(t, err)

	assert.Equal(t, typeDefinition, typeDef)
}

func TestTypeService_GetSecretType_Error(t *testing.T) {
	var service = typeService{
		types: map[string]TypeDefinition{},
	}

	_, err := service.GetSecretType(context.Background(), "secretType")
	require.Error(t, err)

	assert.True(t, errors.Is(err, ErrNotSupportedSecretType))
}
