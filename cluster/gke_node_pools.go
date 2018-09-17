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
	"github.com/banzaicloud/pipeline/internal/providers/google"
	"github.com/banzaicloud/pipeline/model"
	pkgClusterGoogle "github.com/banzaicloud/pipeline/pkg/cluster/gke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	gke "google.golang.org/api/container/v1"
)

//createNodePoolsModelFromRequestData creates an array of GoogleNodePoolModel from the nodePoolsData received through create/update requests
func createNodePoolsModelFromRequestData(nodePoolsData map[string]*pkgClusterGoogle.NodePool, userId uint) ([]*model.GKENodePoolModel, error) {

	nodePoolsCount := len(nodePoolsData)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}
	nodePoolsModel := make([]*model.GKENodePoolModel, nodePoolsCount)

	i := 0
	for nodePoolName, nodePoolData := range nodePoolsData {
		nodePoolsModel[i] = &model.GKENodePoolModel{
			CreatedBy:        userId,
			Name:             nodePoolName,
			Autoscaling:      nodePoolData.Autoscaling,
			NodeMinCount:     nodePoolData.MinCount,
			NodeMaxCount:     nodePoolData.MaxCount,
			NodeCount:        nodePoolData.Count,
			NodeInstanceType: nodePoolData.NodeInstanceType,
		}
		i++
	}

	return nodePoolsModel, nil
}

// createNodePoolsModelFromRequest creates an array of GoogleNodePoolModel from the nodePoolsData received through create/update requests
func createNodePoolsModelFromRequest(nodePoolsData map[string]*pkgClusterGoogle.NodePool, userId uint) ([]*google.GKENodePoolModel, error) {
	nodePoolsCount := len(nodePoolsData)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}
	nodePoolsModel := make([]*google.GKENodePoolModel, nodePoolsCount)

	i := 0
	for nodePoolName, nodePoolData := range nodePoolsData {
		nodePoolsModel[i] = &google.GKENodePoolModel{
			CreatedBy:        userId,
			Name:             nodePoolName,
			Autoscaling:      nodePoolData.Autoscaling,
			NodeMinCount:     nodePoolData.MinCount,
			NodeMaxCount:     nodePoolData.MaxCount,
			NodeCount:        nodePoolData.Count,
			NodeInstanceType: nodePoolData.NodeInstanceType,
		}

		i++
	}

	return nodePoolsModel, nil
}

//createNodePoolsFromClusterModel creates an array of gke NodePool from the given cluster model
func createNodePoolsFromClusterModel(clusterModel *google.GKEClusterModel) ([]*gke.NodePool, error) {
	nodePoolsCount := len(clusterModel.NodePools)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	nodePools := make([]*gke.NodePool, nodePoolsCount)

	for i := 0; i < nodePoolsCount; i++ {
		nodePoolModel := clusterModel.NodePools[i]

		nodePools[i] = &gke.NodePool{
			Name: nodePoolModel.Name,
			Config: &gke.NodeConfig{
				Labels:      map[string]string{pkgCommon.LabelKey: nodePoolModel.Name},
				MachineType: nodePoolModel.NodeInstanceType,
				OauthScopes: []string{
					"https://www.googleapis.com/auth/logging.write",
					"https://www.googleapis.com/auth/monitoring",
					"https://www.googleapis.com/auth/devstorage.read_write",
					"https://www.googleapis.com/auth/cloud-platform",
					"https://www.googleapis.com/auth/compute",
				},
			},
			InitialNodeCount: int64(nodePoolModel.NodeCount),
			Version:          clusterModel.NodeVersion,
		}

		if nodePoolModel.Autoscaling {
			nodePools[i].Autoscaling = &gke.NodePoolAutoscaling{
				Enabled:      true,
				MinNodeCount: int64(nodePoolModel.NodeMinCount),
				MaxNodeCount: int64(nodePoolModel.NodeMaxCount),
			}
		} else {
			nodePools[i].Autoscaling = &gke.NodePoolAutoscaling{
				Enabled: false,
			}
		}

	}

	return nodePools, nil
}

// createNodePoolsRequestDataFromNodePoolModel returns a map of node pool name -> GoogleNodePool from the given nodePoolsModel
func createNodePoolsRequestDataFromNodePoolModel(nodePoolsModel []*google.GKENodePoolModel) (map[string]*pkgClusterGoogle.NodePool, error) {
	nodePoolsCount := len(nodePoolsModel)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	nodePools := make(map[string]*pkgClusterGoogle.NodePool)

	for i := 0; i < nodePoolsCount; i++ {
		nodePoolModel := nodePoolsModel[i]
		nodePools[nodePoolModel.Name] = &pkgClusterGoogle.NodePool{
			Autoscaling:      nodePoolModel.Autoscaling,
			MinCount:         nodePoolModel.NodeMinCount,
			MaxCount:         nodePoolModel.NodeMaxCount,
			Count:            nodePoolModel.NodeCount,
			NodeInstanceType: nodePoolModel.NodeInstanceType,
		}
	}

	return nodePools, nil
}
