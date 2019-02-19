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

package cluster

import (
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/internal/providers/google"
	pkgClusterGoogle "github.com/banzaicloud/pipeline/pkg/cluster/gke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/gin-gonic/gin/json"
	gke "google.golang.org/api/container/v1"
)

const (
	pool1Name             = "pool1"
	pool1Count            = 1
	pool1NodeInstanceType = "instanceType1"
	pool2Name             = "pool2"
	pool2Count            = 2
	pool2NodeInstanceType = "instanceType2"
	nodeVersion           = "gke-1.9"
	labelName             = "custom"
	labelValue            = "value"

	userId = 1
)

// nolint: gochecknoglobals
var (
	clusterModelWithNilNodePools = &google.GKEClusterModel{NodeVersion: nodeVersion}

	clusterModelWithEmptyNodePools = &google.GKEClusterModel{
		NodeVersion: nodeVersion,
		NodePools:   []*google.GKENodePoolModel{},
	}

	clusterModel = &google.GKEClusterModel{
		NodeVersion: nodeVersion,
		NodePools: []*google.GKENodePoolModel{
			{Name: pool1Name, NodeCount: pool1Count, NodeInstanceType: pool1NodeInstanceType},
			{Name: pool2Name, NodeCount: pool2Count, NodeInstanceType: pool2NodeInstanceType},
		},
	}
)

func TestCreateNodePoolsModelFromRequest(t *testing.T) {
	// given
	emptyNodePoolsData := map[string]*pkgClusterGoogle.NodePool{}

	modePoolsData := map[string]*pkgClusterGoogle.NodePool{
		pool1Name: {Count: pool1Count, NodeInstanceType: pool1NodeInstanceType, Labels: map[string]string{labelName: labelValue}},
		pool2Name: {Count: pool2Count, NodeInstanceType: pool2NodeInstanceType, Labels: map[string]string{labelName: labelValue}},
	}

	labels := map[string]string{labelName: labelValue}

	nodePoolsModel := []*google.GKENodePoolModel{
		{CreatedBy: userId, Name: pool1Name, NodeCount: pool1Count, NodeInstanceType: pool1NodeInstanceType, Labels: labels},
		{CreatedBy: userId, Name: pool2Name, NodeCount: pool2Count, NodeInstanceType: pool2NodeInstanceType, Labels: labels},
	}

	testCases := []struct {
		name                   string
		inputNodePoolsData     map[string]*pkgClusterGoogle.NodePool
		expectedNodePoolsModel []*google.GKENodePoolModel
		expectedErr            error
	}{
		{name: "create node pools model from nil", inputNodePoolsData: nil, expectedNodePoolsModel: nil, expectedErr: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create node pools model from empty", inputNodePoolsData: emptyNodePoolsData, expectedNodePoolsModel: nil, expectedErr: pkgErrors.ErrorNodePoolNotProvided},
		{name: "create node pools model", inputNodePoolsData: modePoolsData, expectedNodePoolsModel: nodePoolsModel, expectedErr: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			nodePoolsModel, err := createNodePoolsModelFromRequest(tc.inputNodePoolsData, userId)

			// then
			if tc.expectedErr != err {
				t.Errorf("Expected error %#v, got: %#v", tc.expectedErr, err)
			}

			// we have to compare the actual and expected slices regardless of the order of the elements
			expected := make(map[string]*google.GKENodePoolModel, len(tc.expectedNodePoolsModel))
			for _, nodePool := range tc.expectedNodePoolsModel {
				expected[nodePool.Name] = nodePool
			}

			actual := make(map[string]*google.GKENodePoolModel, len(nodePoolsModel))
			for _, nodePool := range nodePoolsModel {
				actual[nodePool.Name] = nodePool
			}

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("Expected node pools model:\n%v, got:\n%v", tc.expectedNodePoolsModel, nodePoolsModel)
			}
		})
	}
}

func TestCreateNodePoolsFromClusterModel(t *testing.T) {
	// given
	nodeConfig1 := &gke.NodeConfig{
		MachineType: pool1NodeInstanceType,
		OauthScopes: []string{
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring",
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
		},
		Labels: map[string]string{
			pkgCommon.LabelKey: pool1Name,
		},
	}

	nodeConfig2 := &gke.NodeConfig{
		MachineType: pool2NodeInstanceType,
		OauthScopes: []string{
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring",
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
		},
		Labels: map[string]string{
			pkgCommon.LabelKey: pool2Name,
		},
	}
	nodePools := []*gke.NodePool{
		{Name: pool1Name, Autoscaling: &gke.NodePoolAutoscaling{Enabled: false, MinNodeCount: 0, MaxNodeCount: 0}, InitialNodeCount: pool1Count, Version: nodeVersion, Config: nodeConfig1},
		{Name: pool2Name, Autoscaling: &gke.NodePoolAutoscaling{Enabled: false, MinNodeCount: 0, MaxNodeCount: 0}, InitialNodeCount: pool2Count, Version: nodeVersion, Config: nodeConfig2},
	}

	testCases := []struct {
		name         string
		clusterModel *google.GKEClusterModel
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

				t.Error("Expected node pools:\n")
				for _, np := range tc.nodePools {
					data, _ := json.Marshal(*np)
					t.Errorf("%v\n", string(data))
				}

				t.Error("Got:\n")
				for _, np := range nodePools {
					data, _ := json.Marshal(*np)
					t.Errorf("%v\n", string(data))
				}
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
		},
		pool2Name: {
			Count:            pool2Count,
			NodeInstanceType: pool2NodeInstanceType,
		},
	}

	testCases := []struct {
		name                 string
		nodePoolsModel       []*google.GKENodePoolModel
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

// nolint: gochecknoglobals
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
