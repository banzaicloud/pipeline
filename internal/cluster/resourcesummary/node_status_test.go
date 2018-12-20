// Copyright © 2018 Banzai Cloud
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

package resourcesummary

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

func TestGetNodeStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		node   *v1.Node
		status string
	}{
		"ready": {
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   conditionTypeReady,
							Status: readyTrue,
						},
					},
				},
			},
			status: StatusReady,
		},
		"not_ready": {
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   conditionTypeReady,
							Status: readyFalse,
						},
					},
				},
			},
			status: StatusNotReady,
		},
		"unknown": {
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   conditionTypeReady,
							Status: "invalid",
						},
					},
				},
			},
			status: StatusUnknown,
		},
	}

	for name, test := range tests {
		name, test := name, test

		t.Run(name, func(t *testing.T) {
			status := GetNodeStatus(test.node)

			assert.Equal(t, test.status, status)
		})
	}
}
