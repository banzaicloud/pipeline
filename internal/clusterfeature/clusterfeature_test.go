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
	"context"
	"testing"

	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestActivateClusterFeature(t *testing.T) {
	tests := []struct {
		name           string
		clusterId      string
		clusterFeature Feature
		checker        func(*testing.T, interface{})
	}{
		{
			name:      "cluster is not ready",
			clusterId: "notready",
			clusterFeature: Feature{
				Name: "clusterisnotready",
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(error)
				assert.True(t, ok)
				assert.NotNil(t, e)
				assert.EqualError(t, e, "cluster is not ready")
			},
		},
		{
			name:      "feature exists",
			clusterId: "",
			clusterFeature: Feature{
				Name: "existingfeature",
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(error)
				assert.True(t, ok)
				assert.NotNil(t, e)
				assert.EqualError(t, e, "feature already exists")
			},
		},
		{
			name:      "could not persist feature",
			clusterId: "",
			clusterFeature: Feature{
				Name: "failtopersist",
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(error)
				assert.True(t, ok)
				assert.NotNil(t, e)
				assert.EqualError(t, e, "failed to persist feature: persistence error")
			},
		},
		{
			name:      "activation succeeded",
			clusterId: "",
			clusterFeature: Feature{
				Name: "success",
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(error)
				assert.False(t, ok)
				assert.Nil(t, e)
			},
		},
	}

	// setup the service, inject mocks
	featureService := NewClusterFeatureService(
		logur.NewTestLogger(),
		&dummyClusterRepository{},
		&dummyFeatureRepository{
			logger: logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"repo": "featurerepo"}),
		},
		&dummyFeatureManager{})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.checker(t, featureService.Activate(context.Background(), test.clusterId, test.clusterFeature))
		})
	}
}
