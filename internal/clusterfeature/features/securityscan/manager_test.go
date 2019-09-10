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
	"testing"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/stretchr/testify/assert"
)

// TestMakeFeatureManager makes sure the constructor always creates an instance that implements the right interface
// and has the right name
func TestMakeFeatureManager(t *testing.T) {
	var securityScanFeatureManager interface{}
	securityScanFeatureManager = MakeFeatureManager()

	fm, ok := securityScanFeatureManager.(clusterfeature.FeatureManager)

	assert.Truef(t, ok, "the instance must implement the 'clusterfeature.FeatureManager' interface")
	assert.Equal(t, FeatureName, fm.Name(), "the feature manager instance name is invalid")
}

func Test(t *testing.T) {

}
