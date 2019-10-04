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
// UpdateGoogleProperties struct for UpdateGoogleProperties
type UpdateGoogleProperties struct {
	Master UpdateGooglePropertiesMaster `json:"master,omitempty"`
	NodeVersion string `json:"nodeVersion,omitempty"`
	NodePools map[string]UpdateNodePoolsGoogle `json:"nodePools,omitempty"`
}
