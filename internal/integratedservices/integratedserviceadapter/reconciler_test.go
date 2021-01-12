// Copyright © 2021 Banzai Cloud
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

package integratedserviceadapter

import (
	"testing"

	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestLatestVersion(t *testing.T) {
	testData := map[string]struct {
		availableVersions map[string][]string
		currentVersion    string
		expectedVersion   string
		expectError       bool
	}{
		"no versions at all": {
			expectError: true,
		},
		"available versions in order": {
			availableVersions: map[string][]string{
				"1.0.0": {},
				"2.0.0": {},
				"3.0.0": {},
			},
			expectedVersion: "3.0.0",
		},
		"current version should also count": {
			availableVersions: map[string][]string{
				"1.0.0": {},
				"2.0.0": {},
				"3.0.0": {},
			},
			currentVersion:  "4.0.0",
			expectedVersion: "4.0.0",
		},
		"unordered available versions": {
			availableVersions: map[string][]string{
				"1.0.0": {},
				"4.0.0": {},
				"3.0.0": {},
			},
			expectedVersion: "4.0.0",
		},
		"latest version has v prefix": {
			availableVersions: map[string][]string{
				"2.0.0":  {},
				"v3.0.0": {},
				"1.0.0":  {},
			},
			expectedVersion: "v3.0.0",
		},
		"latest version does not have v prefix": {
			availableVersions: map[string][]string{
				"v2.0.0": {},
				"3.0.0":  {},
				"v1.0.0": {},
			},
			expectedVersion: "3.0.0",
		},
		"handle invalid versions": {
			availableVersions: map[string][]string{
				"v2.0.0": {},
				"asd":    {},
				"3.0.0":  {},
				"v1.0.0": {},
			},
			expectedVersion: "3.0.0",
			expectError:     true,
		},
	}

	for name, td := range testData {
		t.Run(name, func(t *testing.T) {
			latest, err := getLatestVersion(v1alpha1.ServiceInstance{
				Status: v1alpha1.ServiceInstanceStatus{
					AvailableVersions: td.availableVersions,
					Version:           td.currentVersion,
				},
			})
			if td.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, td.expectedVersion, latest)
		})
	}
}
