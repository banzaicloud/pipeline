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
// SpotguideDetailsResponse struct for SpotguideDetailsResponse
type SpotguideDetailsResponse struct {
	Name string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Readme string `json:"readme,omitempty"`
	Version string `json:"version,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Resources RequestedResources `json:"resources,omitempty"`
	Questions []SpotguideOption `json:"questions,omitempty"`
}
