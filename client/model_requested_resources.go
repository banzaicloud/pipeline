/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: 0.3.0
 * Contact: info@banzaicloud.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package client

type RequestedResources struct {
	// Total CPU requested for the cluster
	SumCpu int32 `json:"sumCpu,omitempty"`
	// Total memory requested for the cluster (GB)
	SumMem  int32    `json:"sumMem,omitempty"`
	Filters []string `json:"filters,omitempty"`
	// If true, recommended instance types will have a similar size
	SameSize bool `json:"sameSize,omitempty"`
	// Percentage of regular (on-demand) nodes in the recommended cluster
	OnDemandPct int32 `json:"onDemandPct,omitempty"`
	// Minimum number of nodes in the recommended cluster
	MinNodes int32 `json:"minNodes,omitempty"`
	// Maximum number of nodes in the recommended cluster
	MaxNodes int32 `json:"maxNodes,omitempty"`
}
