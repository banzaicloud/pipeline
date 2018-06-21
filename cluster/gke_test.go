package cluster

import (
	"github.com/banzaicloud/pipeline/model"
	pkgClusterGoogle "github.com/banzaicloud/pipeline/pkg/cluster/google"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	gke "google.golang.org/api/container/v1"
	"reflect"
	"testing"
)

const (
	pool1Name             = "pool1"
	pool1Count            = 1
	pool1NodeInstanceType = "instanceType1"
	pool1ServiceAccount   = "service-account-1"
	pool2Name             = "pool2"
	pool2Count            = 2
	pool2NodeInstanceType = "instanceType2"
	pool2ServiceAccount   = "service-account-2"
	nodeVersion           = "gke-1.9"
)

var (
	clusterModelWithNilNodePools = &model.GoogleClusterModel{NodeVersion: nodeVersion}

	clusterModelWithEmptyNodePools = &model.GoogleClusterModel{
		NodeVersion: nodeVersion,
		NodePools:   []*model.GoogleNodePoolModel{},
	}

	clusterModel = &model.GoogleClusterModel{
		NodeVersion: nodeVersion,
		NodePools: []*model.GoogleNodePoolModel{
			{Name: pool1Name, NodeCount: pool1Count, NodeInstanceType: pool1NodeInstanceType, ServiceAccount: pool1ServiceAccount},
			{Name: pool2Name, NodeCount: pool2Count, NodeInstanceType: pool2NodeInstanceType, ServiceAccount: pool2ServiceAccount},
		},
	}
)

func TestCreateNodePoolsModelFromRequestData(t *testing.T) {
	// given
	emptyNodePoolsData := map[string]*pkgClusterGoogle.NodePool{}

	modePoolsData := map[string]*pkgClusterGoogle.NodePool{
		pool1Name: {Count: pool1Count, NodeInstanceType: pool1NodeInstanceType, ServiceAccount: pool1ServiceAccount},
		pool2Name: {Count: pool2Count, NodeInstanceType: pool2NodeInstanceType, ServiceAccount: pool2ServiceAccount},
	}

	nodePoolsModel := []*model.GoogleNodePoolModel{
		{Name: pool1Name, NodeCount: pool1Count, NodeInstanceType: pool1NodeInstanceType, ServiceAccount: pool1ServiceAccount},
		{Name: pool2Name, NodeCount: pool2Count, NodeInstanceType: pool2NodeInstanceType, ServiceAccount: pool2ServiceAccount},
	}

	testCases := []struct {
		name                   string
		inputNodePoolsData     map[string]*pkgClusterGoogle.NodePool
		expectedNodePoolsModel []*model.GoogleNodePoolModel
		expectedErr            error
	}{
		{name: "create node pools model from nil", inputNodePoolsData: nil, expectedNodePoolsModel: nil, expectedErr: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create node pools model from empty", inputNodePoolsData: emptyNodePoolsData, expectedNodePoolsModel: nil, expectedErr: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create node pools model", inputNodePoolsData: modePoolsData, expectedNodePoolsModel: nodePoolsModel, expectedErr: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			nodePoolsModel, err := createNodePoolsModelFromRequestData(tc.inputNodePoolsData)

			// then
			if tc.expectedErr != err {
				t.Errorf("Expected error %#v, got: %#v", tc.expectedErr, err)
			}

			// we have to compare the actual and expected slices regardless of the order of the elements
			expected := make(map[string]*model.GoogleNodePoolModel, len(tc.expectedNodePoolsModel))
			for _, nodePool := range tc.expectedNodePoolsModel {
				expected[nodePool.Name] = nodePool
			}

			actual := make(map[string]*model.GoogleNodePoolModel, len(nodePoolsModel))
			for _, nodePool := range nodePoolsModel {
				actual[nodePool.Name] = nodePool
			}

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("Expected node pools model: %v, got: %v", tc.expectedNodePoolsModel, nodePoolsModel)
			}
		})
	}
}

func TestCreateNodePoolsFromClusterModel(t *testing.T) {
	// given
	nodeConfig1 := &gke.NodeConfig{
		MachineType:    pool1NodeInstanceType,
		ServiceAccount: pool1ServiceAccount,
		OauthScopes: []string{
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring",
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
		},
	}

	nodeConfig2 := &gke.NodeConfig{
		MachineType:    pool2NodeInstanceType,
		ServiceAccount: pool2ServiceAccount,
		OauthScopes: []string{
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring",
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
		},
	}
	nodePools := []*gke.NodePool{
		{Name: pool1Name, Autoscaling: &gke.NodePoolAutoscaling{Enabled: false, MinNodeCount: 0, MaxNodeCount: 0}, InitialNodeCount: pool1Count, Version: nodeVersion, Config: nodeConfig1},
		{Name: pool2Name, Autoscaling: &gke.NodePoolAutoscaling{Enabled: false, MinNodeCount: 0, MaxNodeCount: 0}, InitialNodeCount: pool2Count, Version: nodeVersion, Config: nodeConfig2},
	}

	testCases := []struct {
		name         string
		clusterModel *model.GoogleClusterModel
		nodePools    []*gke.NodePool
		err          error
	}{
		{name: "create node pools from nil model", clusterModel: clusterModelWithNilNodePools, nodePools: nil, err: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create node pools from empty model", clusterModel: clusterModelWithEmptyNodePools, nodePools: nil, err: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create node pools from model", clusterModel: clusterModel, nodePools: nodePools, err: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			nodePools, err := createNodePoolsFromClusterModel(tc.clusterModel)

			// then
			if tc.err != err {
				t.Errorf("Expected error %#v, got: %#v", tc.err, err)
			}

			if !reflect.DeepEqual(tc.nodePools, nodePools) {
				t.Errorf("Expected node pools model: %v, got: %v", tc.nodePools, nodePools)
			}
		})
	}
}

func TestCreateRequestNodePoolsFromNodePoolModel(t *testing.T) {
	// given
	nodePoolsRequestData := map[string]*pkgClusterGoogle.NodePool{
		pool1Name: {
			Count:            pool1Count,
			NodeInstanceType: pool1NodeInstanceType,
			ServiceAccount:   pool1ServiceAccount,
		},
		pool2Name: {
			Count:            pool2Count,
			NodeInstanceType: pool2NodeInstanceType,
			ServiceAccount:   pool2ServiceAccount,
		},
	}

	testCases := []struct {
		name                 string
		nodePoolsModel       []*model.GoogleNodePoolModel
		nodePoolsRequestData map[string]*pkgClusterGoogle.NodePool
		err                  error
	}{
		{name: "create request node pools from node pools model", nodePoolsModel: clusterModel.NodePools, nodePoolsRequestData: nodePoolsRequestData, err: nil},
		{name: "create request node pools from nil model", nodePoolsModel: clusterModelWithNilNodePools.NodePools, nodePoolsRequestData: nil, err: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create request node pools from empty model", nodePoolsModel: clusterModelWithEmptyNodePools.NodePools, nodePoolsRequestData: nil, err: pkgErrors.ErrorNodePoolNotProvided},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			requestNodePools, err := createNodePoolsRequestDataFromNodePoolModel(tc.nodePoolsModel)
			// then
			if tc.err != err {
				t.Errorf("Expected error %#v, got: %#v", tc.err, err)
			}

			if !reflect.DeepEqual(tc.nodePoolsRequestData, requestNodePools) {
				t.Errorf("Expected node pools request data: %v, got: %v", tc.nodePoolsRequestData, requestNodePools)
			}

		})
	}
}

func TestUpdateVersions(t *testing.T) {

	cases := []struct {
		name   string
		input  []string
		output []string
	}{
		{name: "update K8S versions", input: okVersionsIn, output: okVersionsOut},
		{name: "update K8S versions 2", input: okVersionsIn2, output: okVersionsOut2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := updateVersions(tc.input)
			if !reflect.DeepEqual(tc.output, output) {
				t.Errorf("Expected output %v, got: %v", tc.output, output)
			}
		})
	}

}

var (
	okVersionsIn = []string{
		"1.9.7-gke.0",
		"1.9.6-gke.1",
		"1.9.3-gke.0",
		"1.8.12-gke.0",
		"1.8.10-gke.0",
		"1.8.8-gke.0",
		"1.7.15-gke.0",
		"1.7.12-gke.2",
		"1.6.13-gke.1",
		"1.5.7",
	}

	okVersionsIn2 = []string{
		"1.9",
		"1.9.6",
		"1.9.3-gke.0",
		"1.9.3-gke.0",
		"1.8.12-gke.0",
		"1.8",
		"1.8.8-gke.0",
		"1.7.15-gke.0",
		"2",
		"1.6",
		"1.5.7",
	}

	okVersionsOut = []string{
		"1.9",
		"1.9.7-gke.0",
		"1.9.6-gke.1",
		"1.9.3-gke.0",
		"1.8",
		"1.8.12-gke.0",
		"1.8.10-gke.0",
		"1.8.8-gke.0",
		"1.7",
		"1.7.15-gke.0",
		"1.7.12-gke.2",
		"1.6",
		"1.6.13-gke.1",
		"1.5",
		"1.5.7",
	}

	okVersionsOut2 = []string{
		"1.9",
		"1.9.6",
		"1.9.3-gke.0",
		"1.8",
		"1.8.12-gke.0",
		"1.8.8-gke.0",
		"1.7",
		"1.7.15-gke.0",
		"2",
		"1.6",
		"1.5",
		"1.5.7",
	}
)
