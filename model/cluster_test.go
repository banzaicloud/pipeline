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
			{Name: "nodepool1", NodeCount: 2, NodeInstanceType: "n1-standard-1", ServiceAccount: "service-account"},
		},
	}

	gcMultiNodePool := GoogleClusterModel{
		ClusterModelId: 1,
		MasterVersion:  "master-node-ver",
		NodeVersion:    "node-ver",
		NodePools: []*GoogleNodePoolModel{
			{Name: "nodepool1", NodeCount: 2, NodeInstanceType: "n1-standard-1", ServiceAccount: "service-account"},
			{Name: "nodepool2", NodeCount: 1, NodeInstanceType: "n1-standard-2", ServiceAccount: "service-account"},
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
			expected: "Master version: master-node-ver, Node version: node-ver, " +
				"Node pools: [(Name: nodepool1, Instance type: n1-standard-1, Node count: 2, Service account: service-account)]",
		},
		{
			name:               "MultipleNodePools",
			googleClusterModel: gcMultiNodePool,
			expected: "Master version: master-node-ver, Node version: node-ver, " +
				"Node pools: [(Name: nodepool1, Instance type: n1-standard-1, Node count: 2, Service account: service-account) (Name: nodepool2, Instance type: n1-standard-2, Node count: 1, Service account: service-account)]",
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
