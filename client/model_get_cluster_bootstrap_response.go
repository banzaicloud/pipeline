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

// GetClusterBootstrapResponse struct for GetClusterBootstrapResponse
type GetClusterBootstrapResponse struct {
	Token                    string `json:"token,omitempty"`
	DiscoveryTokenCaCertHash string `json:"discoveryTokenCaCertHash,omitempty"`
	MasterAddress            string `json:"masterAddress,omitempty"`
}
