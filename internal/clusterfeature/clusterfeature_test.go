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
		clusterId      uint
		clusterFeature Feature
		checker        func(*testing.T, interface{})
	}{
		{
			name:      "could not select feature",
			clusterId: clusterNotReady,
			clusterFeature: Feature{
				Name: featureSelectionErrorName,
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(unsupportedFeatureError)
				assert.True(t, ok)
				assert.NotNil(t, e)
				assert.EqualError(t, e, "Feature: feature-couldnotselect, Message: could not select feature")
			},
		},
		{
			name:      "feature already exists",
			clusterId: clusterNotReady,
			clusterFeature: Feature{
				Name: featureExists,
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(featureExistsError)
				assert.True(t, ok)
				assert.NotNil(t, e)
				assert.EqualError(t, e, "Feature: existing-feature, Message: feature already exists")
			},
		},
		{
			name:      "cluster is not ready",
			clusterId: clusterNotReady,
			clusterFeature: Feature{
				Name: "clusterisnotready",
				Spec: nil,
			},
			checker: func(t *testing.T, response interface{}) {
				e, ok := response.(clusterNotReadyError)
				assert.True(t, ok)
				assert.NotNil(t, e)
				assert.EqualError(t, e, "Feature: clusterisnotready, Message: cluster is not ready")
			},
		},
		{
			name:      "could not persist feature",
			clusterId: clusterReady,
			clusterFeature: Feature{
				Name: featureCouldNotPersist,
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
			clusterId: clusterReady,
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

	// override the feature selector
	featureService.featureSelector = &dummyFeatureSelector{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.checker(t, featureService.Activate(context.Background(), test.clusterId, test.clusterFeature.Name, test.clusterFeature.Spec))
		})
	}
}

func testClusterFeature(t *testing.T) {
	t.Log("do nothing")
	//lr := logrus.New()
	//l := logur.WithFields(logrusadapter.New(lr), map[string]interface{}{"app": "clusterfeature-iTest"})
	//db := config.DB()
	//
	//secretValidator := providers.NewSecretValidator(secret.Store)
	//cm := cluster.NewManager(cluster2.NewClusters(config.DB()), secretValidator, cluster.NewNopClusterEvents(), nil, nil, nil, lr, logur.NewErrorHandler(l))
	//
	//cr := clusterfeatureadapter.NewClusterService(cm)
	//fr := clusterfeatureadapter.NewGormFeatureRepository(db)
	//fm := clusterfeatureadapter.NewExternalDnsFeatureManager(cr)
	//
	//cps := NewClusterFeatureService(l, cr, fr, fm)
	//
	//if err := cps.Activate(context.Background(), 3, "", map[string]interface{}{}); err != nil {
	//	t.Error(err)
	//}

}
