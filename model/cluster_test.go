package model

import (
	"fmt"
	"testing"
)

func TestGoogleClusterModelStringer(t *testing.T) {
	// given
	gcSingleNodePool := GoogleClusterModel{
		ClusterModelId: 1,
		MasterVersion:  "master-node-ver",
		NodeVersion:    "node-ver",
		NodePools: []*GoogleNodePoolModel{
			{
				ID:               1,
				CreatedBy:        1,
				ClusterModelId:   1,
				Name:             "nodepool1",
				Autoscaling:      false,
				NodeMinCount:     1,
				NodeMaxCount:     2,
				NodeCount:        2,
				NodeInstanceType: "n1-standard-1",
				ServiceAccount:   "service-account",
				Delete:           false,
			},
		},
	}

	gcMultiNodePool := GoogleClusterModel{
		ClusterModelId: 1,
		MasterVersion:  "master-node-ver",
		NodeVersion:    "node-ver",
		NodePools: []*GoogleNodePoolModel{
			{
				ID:               1,
				CreatedBy:        1,
				Name:             "nodepool1",
				Autoscaling:      false,
				NodeMinCount:     1,
				NodeMaxCount:     2,
				NodeCount:        2,
				NodeInstanceType: "n1-standard-1",
				ServiceAccount:   "service-account",
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
				ServiceAccount:   "service-account",
			},
		},
	}

	cases := []struct {
		name               string
		googleClusterModel GoogleClusterModel
		expected           string
	}{
		{
			name:               "SingleNodePools",
			googleClusterModel: gcSingleNodePool,
			expected:           "Master version: master-node-ver, Node version: node-ver, Node pools: [ID: 1, createdAt: 0001-01-01 00:00:00 +0000 UTC, createdBy: 1, Name: nodepool1, Autoscaling: false, NodeMinCount: 1, NodeMaxCount: 2, NodeCount: 2, ServiceAccount: service-account]",
		},
		{
			name:               "MultipleNodePools",
			googleClusterModel: gcMultiNodePool,
			expected:           "Master version: master-node-ver, Node version: node-ver, Node pools: [ID: 1, createdAt: 0001-01-01 00:00:00 +0000 UTC, createdBy: 1, Name: nodepool1, Autoscaling: false, NodeMinCount: 1, NodeMaxCount: 2, NodeCount: 2, ServiceAccount: service-account ID: 1, createdAt: 0001-01-01 00:00:00 +0000 UTC, createdBy: 1, Name: nodepool2, Autoscaling: false, NodeMinCount: 1, NodeMaxCount: 1, NodeCount: 1, ServiceAccount: service-account]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			actual := fmt.Sprint(tc.googleClusterModel)

			// then
			if actual != tc.expected {
				t.Errorf("Expected: %q, got: %q", tc.expected, actual)
			}
		})
	}

}
