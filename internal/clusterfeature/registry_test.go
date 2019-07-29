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

package clusterfeature

import (
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureRegistry_GetFeatureManager(t *testing.T) {
	expectedFeatureManager := &dummyFeatureManager{}

	registry := NewFeatureRegistry(map[string]FeatureManager{
		"myFeature": expectedFeatureManager,
	})

	featureManager, err := registry.GetFeatureManager("myFeature")
	require.NoError(t, err)

	assert.Equal(t, featureManager.Name(), "myFeature")
}

func TestFeatureRegistry_GetFeatureManager_UnknownFeature(t *testing.T) {
	registry := NewFeatureRegistry(map[string]FeatureManager{})

	featureManager, err := registry.GetFeatureManager("myFeature")
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownFeatureError{}))
	assert.True(t, errors.Is(err, UnknownFeatureError{FeatureName: "myFeature"}))

	assert.Nil(t, featureManager)
}
