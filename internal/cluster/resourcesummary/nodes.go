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
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// CalculateNodesTotalCapacityAndAllocatable calculates capacity and allocatable resources of the given nodes.
func CalculateNodesTotalCapacityAndAllocatable(nodes []v1.Node) (map[v1.ResourceName]resource.Quantity, map[v1.ResourceName]resource.Quantity) {
	caps, allocs := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, node := range nodes {
		nodeCaps, nodeAllocs := NodeCapacityAndAllocatable(node)

		for nodeCapName, nodeCapValue := range nodeCaps {
			if value, ok := caps[nodeCapName]; !ok {
				caps[nodeCapName] = *nodeCapValue.Copy()
			} else {
				value.Add(nodeCapValue)
				caps[nodeCapName] = value
			}
		}

		for nodeAllocName, nodeAllocValue := range nodeAllocs {
			if value, ok := allocs[nodeAllocName]; !ok {
				allocs[nodeAllocName] = *nodeAllocValue.Copy()
			} else {
				value.Add(nodeAllocValue)
				allocs[nodeAllocName] = value
			}
		}
	}

	return caps, allocs
}

// NodeCapacityAndAllocatable calculates capacity and allocatable resources for a node.
func NodeCapacityAndAllocatable(node v1.Node) (map[v1.ResourceName]resource.Quantity, map[v1.ResourceName]resource.Quantity) {
	capacity := map[v1.ResourceName]resource.Quantity{
		v1.ResourceCPU:    {},
		v1.ResourceMemory: {},
	}

	allocatable := map[v1.ResourceName]resource.Quantity{
		v1.ResourceCPU:    {},
		v1.ResourceMemory: {},
	}

	if cpu := node.Status.Capacity.Cpu(); cpu != nil {
		capacity[v1.ResourceCPU] = *cpu
	}

	if cpu := node.Status.Allocatable.Cpu(); cpu != nil {
		allocatable[v1.ResourceCPU] = *cpu
	}

	if mem := node.Status.Capacity.Memory(); mem != nil {
		capacity[v1.ResourceMemory] = *mem
	}

	if mem := node.Status.Allocatable.Memory(); mem != nil {
		allocatable[v1.ResourceMemory] = *mem
	}

	return capacity, allocatable
}
