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

package anchore

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate mockery -name ConfigProvider -inpkg -testonly

func TestConfigProviderChain_GetConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("first_not_found", func(t *testing.T) {
		t.Parallel()

		provider1 := new(MockConfigProvider)
		provider2 := new(MockConfigProvider)

		clusterID := uint(1)
		ctx := context.Background()
		expectedConfig := Config{
			Endpoint: "https://example.anchore.com",
			User:     "user",
			Password: "password",
		}

		provider1.On("GetConfiguration", ctx, clusterID).Return(Config{}, ErrConfigNotFound)
		provider2.On("GetConfiguration", ctx, clusterID).Return(expectedConfig, nil)

		chain := ConfigProviderChain{provider1, provider2}

		config, err := chain.GetConfiguration(ctx, clusterID)
		require.NoError(t, err)

		assert.Equal(t, expectedConfig, config)
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})

	t.Run("first_error", func(t *testing.T) {
		t.Parallel()

		provider1 := new(MockConfigProvider)
		provider2 := new(MockConfigProvider)

		clusterID := uint(1)
		ctx := context.Background()
		e := errors.New("error")

		provider1.On("GetConfiguration", ctx, clusterID).Return(Config{}, e)

		chain := ConfigProviderChain{provider1, provider2}

		_, err := chain.GetConfiguration(ctx, clusterID)
		require.Error(t, err)

		assert.Equal(t, e, err)
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		provider1 := new(MockConfigProvider)
		provider2 := new(MockConfigProvider)

		clusterID := uint(1)
		ctx := context.Background()

		provider1.On("GetConfiguration", ctx, clusterID).Return(Config{}, ErrConfigNotFound)
		provider2.On("GetConfiguration", ctx, clusterID).Return(Config{}, ErrConfigNotFound)

		chain := ConfigProviderChain{provider1, provider2}

		_, err := chain.GetConfiguration(ctx, clusterID)
		require.Error(t, err)

		assert.Equal(t, ErrConfigNotFound, err)
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})
}

func TestStaticConfigProvider_GetConfiguration(t *testing.T) {
	expectedConfig := Config{
		Endpoint: "https://example.anchore.com",
		User:     "user",
		Password: "password",
	}

	provider := StaticConfigProvider{
		Config: expectedConfig,
	}

	config, err := provider.GetConfiguration(context.Background(), 1)
	require.NoError(t, err)

	assert.Equal(t, expectedConfig, config)
}
