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
	"encoding/json"
	"testing"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/stretchr/testify/assert"
	"logur.dev/logur"
)

func TestMakeFeatureOperator(t *testing.T) {
	var ssFeatureOperator interface{}
	ssFeatureOperator = MakeFeatureOperator(
		clusterGetterMock{},
		clusterServiceMock{},
		helmServiceMock{},
		secretStoreMock{},
		commonadapter.NewLogger(logur.NewTestLogger()),
	)

	fo, ok := ssFeatureOperator.(clusterfeature.FeatureOperator)

	assert.Truef(t, ok, "the instance must implement the 'clusterfeature.FeatureOperator' interface")
	assert.Equal(t, FeatureName, fo.Name(), "the feature manager instance name is invalid")
}

func TestFeatureOperator_ProcessChartValues(t *testing.T) {
	ssFeatureOperator := MakeFeatureOperator(
		clusterGetterMock{},
		clusterServiceMock{},
		helmServiceMock{},
		secretStoreMock{},
		commonadapter.NewLogger(logur.NewTestLogger()),
	)

	//anchoreSecretValues := map[string]string{
	//	"username": "test_username",
	//	"password": "test_password",
	//}

	values, err := ssFeatureOperator.processChartValues(context.Background(), 10, "test_username")
	assert.Nil(t, err, "failed to process chart values ")
	assert.NotNil(t, values, "values should be filled")

	// validate the processed values
	var ssValues SecurityScanChartValues
	err = json.Unmarshal(values, &ssValues)

	assert.Nil(t, err, "failed to unmarshal values")
	assert.NotNil(t, ssValues, "could not unmarshal values")
	assert.NotNil(t, ssValues.Anchore, "anchore values lost")
	assert.Equal(t, "test_username", ssValues.Anchore.User, "anchore user lost during transformation")
	assert.Equal(t, "test_password", ssValues.Anchore.Password, "anchore password lost during transformation")

}
