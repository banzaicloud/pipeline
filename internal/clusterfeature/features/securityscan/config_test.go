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

package securityscan

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/securityscan"
)

// TODO: replace mock with in-memory implementation?
//go:generate mockery -dir $PWD/internal/common -name SecretStore -testonly -output $PWD/internal/clusterfeature/features/securityscan -outpkg securityscan

func TestAnchoreConfigProvider_GetConfiguration_CustomAnchore(t *testing.T) {
	featureRepository := clusterfeature.NewInMemoryFeatureRepository(map[uint][]clusterfeature.Feature{
		1: {
			{
				Name: "securityscan",
				Spec: map[string]interface{}{
					"customAnchore": map[string]interface{}{
						"enabled":  true,
						"url":      "https://anchore.example.com",
						"secretId": "secretId",
					},
				},
				Output: nil,
				Status: clusterfeature.FeatureStatusActive,
			},
		},
	})

	secretStore := new(SecretStore)
	secretStore.On("GetSecretValues", mock.Anything, "secretId").Return(
		map[string]string{
			"username": "user",
			"password": "password",
		},
		nil,
	)

	configProvider := NewAnchoreConfigProvider(nil, featureRepository, secretStore, common.NewNoopLogger())

	config, err := configProvider.GetConfiguration(context.Background(), 1)
	require.NoError(t, err)

	assert.Equal(
		t,
		securityscan.AnchoreConfig{
			Endpoint: "https://anchore.example.com",
			User:     "user",
			Password: "password",
		},
		config,
	)

	secretStore.AssertExpectations(t)
}

func TestAnchoreConfigProvider_GetConfiguration_GlobalAnchore(t *testing.T) {
	featureRepository := clusterfeature.NewInMemoryFeatureRepository(map[uint][]clusterfeature.Feature{
		1: {
			{
				Name:   "securityscan",
				Spec:   map[string]interface{}{},
				Output: nil,
				Status: clusterfeature.FeatureStatusActive,
			},
		},
	})

	secretStore := new(SecretStore)

	globalConfig := securityscan.AnchoreConfig{
		Endpoint: "https://anchore.example.com",
		User:     "user",
		Password: "password",
	}

	configProvider := NewAnchoreConfigProvider(&globalConfig, featureRepository, secretStore, common.NewNoopLogger())

	config, err := configProvider.GetConfiguration(context.Background(), 1)
	require.NoError(t, err)

	assert.Equal(t, globalConfig, config)

	secretStore.AssertExpectations(t)
}

func TestAnchoreConfigProvider_GetConfiguration_NoAnchore(t *testing.T) {
	featureRepository := clusterfeature.NewInMemoryFeatureRepository(map[uint][]clusterfeature.Feature{
		1: {
			{
				Name:   "securityscan",
				Spec:   map[string]interface{}{},
				Output: nil,
				Status: clusterfeature.FeatureStatusActive,
			},
		},
	})

	secretStore := new(SecretStore)

	configProvider := NewAnchoreConfigProvider(nil, featureRepository, secretStore, common.NewNoopLogger())

	_, err := configProvider.GetConfiguration(context.Background(), 1)
	require.Error(t, err)

	secretStore.AssertExpectations(t)
}
