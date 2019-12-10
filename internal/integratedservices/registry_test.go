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

package integratedservices

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegratedServiceManagerRegistry_GetIntegratedServiceManager(t *testing.T) {
	expectedIntegratedSErviceManager := dummyIntegratedServiceManager{
		TheName: "myIntegratedService",
	}

	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{
		expectedIntegratedSErviceManager,
	})

	integratedServiceManager, err := registry.GetIntegratedServiceManager("myIntegratedService")
	require.NoError(t, err)

	assert.Equal(t, expectedIntegratedSErviceManager, integratedServiceManager)
}

func TestIntegratedServiceManagerRegistry_GetIntegratedServiceManager_UnknownIntegratedService(t *testing.T) {
	registry := MakeIntegratedServiceManagerRegistry([]IntegratedServiceManager{})

	integratedServiceManager, err := registry.GetIntegratedServiceManager("myIntegratedService")
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownIntegratedServiceError{}))
	assert.True(t, errors.Is(err, UnknownIntegratedServiceError{IntegratedServiceName: "myIntegratedService"}))

	assert.Nil(t, integratedServiceManager)
}

func TestIntegratedServiceNameOperatorRegistry_GetIntegratedServiceOperator(t *testing.T) {
	expectedIntegratedServiceOperator := dummyIntegratedServiceOperator{
		TheName: "myIntegratedService",
	}

	registry := MakeIntegratedServiceOperatorRegistry([]IntegratedServiceOperator{
		expectedIntegratedServiceOperator,
	})

	integratedServiceOperator, err := registry.GetIntegratedServiceOperator("myIntegratedService")
	require.NoError(t, err)

	assert.Equal(t, expectedIntegratedServiceOperator, integratedServiceOperator)
}

func TestIntegratedServiceOperatorRegistry_GetIntegratedServiceOperator_UnknownIntegratedService(t *testing.T) {
	registry := MakeIntegratedServiceOperatorRegistry([]IntegratedServiceOperator{})

	integratedServiceOperator, err := registry.GetIntegratedServiceOperator("myIntegratedService")
	require.Error(t, err)

	assert.True(t, errors.As(err, &UnknownIntegratedServiceError{}))
	assert.True(t, errors.Is(err, UnknownIntegratedServiceError{IntegratedServiceName: "myIntegratedService"}))

	assert.Nil(t, integratedServiceOperator)
}

type dummyIntegratedServiceManager struct {
	PassthroughIntegratedServiceSpecPreparer

	TheName         string
	Output          IntegratedServiceOutput
	ValidationError error
}

func (d dummyIntegratedServiceManager) Name() string {
	return d.TheName
}

func (d dummyIntegratedServiceManager) GetOutput(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) (IntegratedServiceOutput, error) {
	return d.Output, nil
}

func (d dummyIntegratedServiceManager) ValidateSpec(ctx context.Context, spec IntegratedServiceSpec) error {
	return d.ValidationError
}

type dummyIntegratedServiceOperator struct {
	TheName string
}

func (d dummyIntegratedServiceOperator) Name() string {
	return d.TheName
}

func (d dummyIntegratedServiceOperator) Apply(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) error {
	return nil
}

func (d dummyIntegratedServiceOperator) Deactivate(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) error {
	return nil
}
