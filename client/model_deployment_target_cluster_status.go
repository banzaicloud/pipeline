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

// DeploymentTargetClusterStatus struct for DeploymentTargetClusterStatus
type DeploymentTargetClusterStatus struct {
	Cloud        string `json:"cloud,omitempty"`
	ClusterId    int32  `json:"clusterId,omitempty"`
	ClusterName  string `json:"clusterName,omitempty"`
	Distribution string `json:"distribution,omitempty"`
	Error        string `json:"error,omitempty"`
	Stale        bool   `json:"stale,omitempty"`
	Status       string `json:"status,omitempty"`
	Version      string `json:"version,omitempty"`
}
