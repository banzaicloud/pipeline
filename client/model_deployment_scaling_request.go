/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments.
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

// DeploymentScalingRequest struct for DeploymentScalingRequest
type DeploymentScalingRequest struct {
	ScaleTarget   string                  `json:"scaleTarget"`
	MinReplicas   int32                   `json:"minReplicas"`
	MaxReplicas   int32                   `json:"maxReplicas"`
	Cpu           ResourceMetric          `json:"cpu,omitempty"`
	Memory        ResourceMetric          `json:"memory,omitempty"`
	CustomMetrics map[string]CustomMetric `json:"customMetrics,omitempty"`
}
