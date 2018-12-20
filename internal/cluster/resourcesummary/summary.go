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
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	zeroCPU    = "0 CPU"
	zeroMemory = "0 B"
)

// Summary describes a resource summary with CPU and Memory capacity/request/limit/allocatable.
type Summary struct {
	CPU    Resource
	Memory Resource
}

// NewSummary creates a new Summary with default zero values.
func NewSummary() Summary {
	return Summary{
		CPU: Resource{
			Capacity:    zeroCPU,
			Allocatable: zeroCPU,
			Limit:       zeroCPU,
			Request:     zeroCPU,
		},
		Memory: Resource{
			Capacity:    zeroMemory,
			Allocatable: zeroMemory,
			Limit:       zeroMemory,
			Request:     zeroMemory,
		},
	}
}

// Resource describes a resource with capacity/request/limit/allocatable.
type Resource struct {
	Capacity    string
	Allocatable string
	Limit       string
	Request     string
}

// GetTotalSummary calculates all resource summary for a cluster.
func GetTotalSummary(client kubernetes.Interface) (*Summary, error) {
	nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}

	podList, err := client.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pods")
	}

	requests, limits := CalculatePodsTotalRequestsAndLimits(podList.Items)
	capacity, allocatable := CalculateNodesTotalCapacityAndAllocatable(nodeList.Items)

	summary := GetSummary(capacity, allocatable, requests, limits)

	return &summary, nil
}

// GetSummary returns Summary type with the given data.
func GetSummary(capacity, allocatable, requests, limits map[v1.ResourceName]resource.Quantity) Summary {
	summary := NewSummary()

	if cpu, ok := capacity[v1.ResourceCPU]; ok {
		summary.CPU.Capacity = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &cpu)
	}

	if cpu, ok := allocatable[v1.ResourceCPU]; ok {
		summary.CPU.Allocatable = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &cpu)
	}

	if value, ok := requests[v1.ResourceCPU]; ok {
		summary.CPU.Request = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &value)
	}

	if value, ok := limits[v1.ResourceCPU]; ok {
		summary.CPU.Limit = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &value)
	}

	if memory, ok := capacity[v1.ResourceMemory]; ok {
		summary.Memory.Capacity = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &memory)
	}

	if memory, ok := allocatable[v1.ResourceMemory]; ok {
		summary.Memory.Allocatable = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &memory)
	}

	if value, ok := requests[v1.ResourceMemory]; ok {
		summary.Memory.Request = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &value)
	}

	if value, ok := limits[v1.ResourceMemory]; ok {
		summary.Memory.Limit = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &value)
	}

	return summary
}
