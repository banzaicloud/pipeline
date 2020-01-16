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
	"fmt"

	"emperror.dev/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

// NodeSummary describes a node's resource summary with CPU and Memory capacity/request/limit/allocatable.
type NodeSummary struct {
	Summary

	Status string
}

// GetNodeSummary returns resource summary for the given node.
func GetNodeSummary(client kubernetes.Interface, node v1.Node) (*NodeSummary, error) {
	fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%s", node.Name))
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "cannot parse field selector for node", "node", node.Name)
	}

	podList, err := client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "cannot parse list pods for node", "node", node.Name)
	}

	requests, limits := CalculatePodsTotalRequestsAndLimits(podList.Items)

	summary := CalculateNodeSummary(node, requests, limits)

	return &summary, nil
}

// CalculateNodeSummary returns NodeSummary type with the given data.
func CalculateNodeSummary(node v1.Node, requests, limits map[v1.ResourceName]resource.Quantity) NodeSummary {
	capacity, allocatable := NodeCapacityAndAllocatable(node)

	summary := NodeSummary{
		Summary: GetSummary(capacity, allocatable, requests, limits),
		Status:  GetNodeStatus(node),
	}

	return summary
}
