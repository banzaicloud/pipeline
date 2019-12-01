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

package google

import (
	"fmt"
	"testing"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
)

func TestGKEClusterModel_String(t *testing.T) {
	cases := []struct {
		name               string
		googleClusterModel GKEClusterModel
		expected           string
	}{
		{
			name: "SingleNodePools",
			googleClusterModel: GKEClusterModel{
				Cluster: clusteradapter.ClusterModel{
					ID:           1,
					Cloud:        "google",
					Distribution: "gke",
				},
				MasterVersion: "master-node-ver",
				NodeVersion:   "node-ver",
				NodePools: []*GKENodePoolModel{
					{
						ID:               1,
						CreatedBy:        1,
						ClusterID:        1,
						Name:             "nodepool1",
						Autoscaling:      false,
						NodeMinCount:     1,
						NodeMaxCount:     2,
						NodeCount:        2,
						NodeInstanceType: "n1-standard-1",
						Delete:           false,
					},
				},
			},
			expected: "Id: 1, Creation date: 0001-01-01 00:00:00 +0000 UTC, Cloud: google, Distribution: gke, Master version: master-node-ver, Node version: node-ver, Node pools: [ID: 1, createdAt: 0001-01-01 00:00:00 +0000 UTC, createdBy: 1, Name: nodepool1, Autoscaling: false, NodeMinCount: 1, NodeMaxCount: 2, NodeCount: 2]",
		},
		{
			name: "MultipleNodePools",
			googleClusterModel: GKEClusterModel{
				Cluster: clusteradapter.ClusterModel{
					ID:           1,
					Cloud:        "google",
					Distribution: "gke",
				},
				MasterVersion: "master-node-ver",
				NodeVersion:   "node-ver",
				NodePools: []*GKENodePoolModel{
					{
						ID:               1,
						CreatedBy:        1,
						Name:             "nodepool1",
						Autoscaling:      false,
						NodeMinCount:     1,
						NodeMaxCount:     2,
						NodeCount:        2,
						NodeInstanceType: "n1-standard-1",
					},
					{
						ID:               1,
						CreatedBy:        1,
						Name:             "nodepool2",
						Autoscaling:      false,
						NodeMinCount:     1,
						NodeMaxCount:     1,
						NodeCount:        1,
						NodeInstanceType: "n1-standard-2",
					},
				},
			},
			expected: "Id: 1, Creation date: 0001-01-01 00:00:00 +0000 UTC, Cloud: google, Distribution: gke, Master version: master-node-ver, Node version: node-ver, Node pools: [ID: 1, createdAt: 0001-01-01 00:00:00 +0000 UTC, createdBy: 1, Name: nodepool1, Autoscaling: false, NodeMinCount: 1, NodeMaxCount: 2, NodeCount: 2 ID: 1, createdAt: 0001-01-01 00:00:00 +0000 UTC, createdBy: 1, Name: nodepool2, Autoscaling: false, NodeMinCount: 1, NodeMaxCount: 1, NodeCount: 1]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := fmt.Sprint(tc.googleClusterModel)

			if actual != tc.expected {
				t.Errorf("Expected: %q, got: %q", tc.expected, actual)
			}
		})
	}
}
