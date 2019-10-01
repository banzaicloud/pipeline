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

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// TestMakeFeatureManager makes sure the constructor always creates an instance that implements the right interface
// and has the right name
func TestMakeFeatureManager(t *testing.T) {
	var securityScanFeatureManager interface{}
	securityScanFeatureManager = MakeFeatureManager(nil)

	fm, ok := securityScanFeatureManager.(clusterfeature.FeatureManager)

	assert.Truef(t, ok, "the instance must implement the 'clusterfeature.FeatureManager' interface")
	assert.Equal(t, FeatureName, fm.Name(), "the feature manager instance name is invalid")
}

// todo add test more cases for validating the spec
func TestFeatureManager_ValidateSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    clusterfeature.FeatureSpec
		checker func(err error) bool
	}{
		{
			name: "initial test case",
			spec: clusterfeature.FeatureSpec{
				"customAnchore": obj{
					"enabled":  true,
					"url":      "anchore.example.com", //mandatory
					"secretId": "mysecretid",          // mandatory
				},
				"policy": obj{
					"policyId": "myPolicyID, select, from backend",
				},
				"releaseWhiteList": []obj{ //optional
					{
						"name":   "name of release 1",                        //mandatory
						"reason": "reason of whitelisting",                   //mandatory
						"regexp": "whitelisted-[0-1]{2}.[a-z]{2,3}-releases", // optional
					},
					{
						"name":   "name of release 2",
						"reason": "reason of whitelisting",
						"regexp": "whitelisted-[0-1]{2}.[a-z]{2,3}-releases",
					},
				},
				"webhookConfig": obj{
					"enabled":    true,                 //
					"selector":   "include or exclude", //mandatory
					"namespaces": []string{"default", "test"},
				},
			},
			checker: func(err error) bool {
				return false
			},
		},
		// todo add more test fixtures here
	}

	ctx := context.Background()
	securityScanFeatureManager := MakeFeatureManager(nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := securityScanFeatureManager.ValidateSpec(ctx, test.spec)
			if err != nil {
				t.Errorf("test failed with errors: %v", err)
			}
		})
	}
}
