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

package api

import (
	"github.com/banzaicloud/pipeline/internal/cluster/resourcesummary"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

func getResourceSummary(capacity, allocatable, requests, limits map[v1.ResourceName]resource.Quantity) *pkgCluster.ResourceSummary {
	summary := resourcesummary.GetSummary(capacity, allocatable, requests, limits)

	return &pkgCluster.ResourceSummary{
		Cpu: &pkgCluster.CPU{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem(summary.CPU),
		},
		Memory: &pkgCluster.Memory{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem(summary.Memory),
		},
	}
}

func getNodeResourceSummary(client kubernetes.Interface, node v1.Node) (*pkgCluster.ResourceSummary, error) {
	summary, err := resourcesummary.GetNodeSummary(client, node)
	if err != nil {
		return nil, err
	}

	return &pkgCluster.ResourceSummary{
		Cpu: &pkgCluster.CPU{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem(summary.CPU),
		},
		Memory: &pkgCluster.Memory{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem(summary.Memory),
		},
		Status: summary.Status,
	}, nil
}
