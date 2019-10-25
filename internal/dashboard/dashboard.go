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

package dashboard

import (
	"time"
)

type Allocatable struct {
	Cpu              string `json:"cpu"`
	EphemeralStorage string `json:"ephemeralStorage"`
	Memory           string `json:"memory"`
	Pods             int64  `json:"pods"`
}

type Capacity struct {
	Cpu              string `json:"cpu"`
	EphemeralStorage string `json:"ephemeralStorage"`
	Memory           string `json:"memory"`
	Pods             int64  `json:"pods"`
}

type Node struct {
	Name              string  `json:"name"`
	CreationTimestamp string  `json:"creationTimestamp"`
	Status            *Status `json:"status"`
}

type Status struct {
	Capacity                    *Capacity    `json:"capacity"`
	Allocatable                 *Allocatable `json:"allocatable"`
	Ready                       string       `json:"ready"`
	LastHeartbeatTime           string       `json:"lastHeartbeatTime"`
	FrequentUnregisterNetDevice string       `json:"frequentUnregisterNetDevice"`
	KernelDeadlock              string       `json:"kernelDeadlock"`
	NetworkUnavailable          string       `json:"networkUnavailable"`
	OutOfDisk                   string       `json:"outOfDisk"`
	MemoryPressure              string       `json:"memoryPressure"`
	DiskPressure                string       `json:"diskPressure"`
	PIDPressure                 string       `json:"pidPressure"`
	CpuUsagePercent             float64      `json:"cpuUsagePercent"`
	StorageUsagePercent         float64      `json:"storageUsagePercent"`
	MemoryUsagePercent          float64      `json:"memoryUsagePercent"`
	InstanceType                string       `json:"instanceType"`
}

// NodePool describes a cluster's node pool.
type NodePool struct {
	Autoscaling     bool                           `json:"autoscaling"`
	Count           int                            `json:"count,omitempty"`
	InstanceType    string                         `json:"instanceType,omitempty"`
	SpotPrice       string                         `json:"spotPrice,omitempty"`
	Preemptible     bool                           `json:"preemptible,omitempty"`
	MinCount        int                            `json:"minCount,omitempty"`
	MaxCount        int                            `json:"maxCount,omitempty"`
	Image           string                         `json:"image,omitempty"`
	Version         string                         `json:"version,omitempty"`
	Labels          map[string]string              `json:"labels,omitempty"`
	ResourceSummary map[string]NodeResourceSummary `json:"resourceSummary,omitempty"`
	CreatedAt       time.Time                      `json:"createdAt,omitempty"`
	CreatorName     string                         `json:"creatorName,omitempty"`
	CreatorID       uint                           `json:"creatorId,omitempty"`
}

type ClusterInfo struct {
	Name                string              `json:"name"`
	Id                  string              `json:"id"`
	Status              string              `json:"status"`
	StatusMessage       string              `json:"statusMessage"`
	Cloud               string              `json:"cloud"`
	Distribution        string              `json:"distribution"`
	Region              string              `json:"region"`
	Location            string              `json:"location"`
	Endpoint            string              `json:"endpoint,omitempty"`
	MasterVersion       string              `json:"masterVersion,omitempty"`
	Project             string              `json:"project,omitempty"`
	ResourceGroup       string              `json:"resourceGroup,omitempty"`
	ClusterGroup        string              `json:"clusterGroup,omitempty"`
	SecretName          string              `json:"secretName,omitempty"`
	Nodes               []Node              `json:"nodes"`
	NodePools           map[string]NodePool `json:"nodePools,omitempty"`
	CreatedAt           time.Time           `json:"createdAt,omitempty"`
	CreatorName         string              `json:"creatorName,omitempty"`
	CreatorId           uint                `json:"creatorId,omitempty"`
	CpuUsagePercent     float64             `json:"cpuUsagePercent"`
	StorageUsagePercent float64             `json:"storageUsagePercent"`
	MemoryUsagePercent  float64             `json:"memoryUsagePercent"`
	Logging             bool                `json:"logging"`
	Monitoring          bool                `json:"monitoring"`
	SecurityScan        bool                `json:"securityscan"`
}

// GetDashboardResponse Api object to be mapped to Get dashboard request
// swagger:model GetDashboardResponse
type GetDashboardResponse struct {
	Clusters []ClusterInfo `json:"clusters"`
}

// GetProviderPathParams is a placeholder for the GetDashboard route path parameters
// swagger:parameters GetDashboard
type GetDashboardPathParams struct {
	// in:path
	OrgId string `json:"orgid"`
}

// ResourceSummary describes a node's resource summary with CPU and Memory capacity/request/limit/allocatable
type ResourceSummary struct {
	CPU    *Resource `json:"cpu,omitempty"`
	Memory *Resource `json:"memory,omitempty"`
}

type NodeResourceSummary struct {
	ResourceSummary

	Status string `json:"status,omitempty"`
}

// Resource describes a resource summary with capacity/request/limit/allocatable
type Resource struct {
	Capacity    string `json:"capacity,omitempty"`
	Allocatable string `json:"allocatable,omitempty"`
	Limit       string `json:"limit,omitempty"`
	Request     string `json:"request,omitempty"`
}
