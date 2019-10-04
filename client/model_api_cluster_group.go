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
// ApiClusterGroup struct for ApiClusterGroup
type ApiClusterGroup struct {
	EnabledFeatures []string `json:"enabledFeatures,omitempty"`
	Id int32 `json:"id,omitempty"`
	Members []ApiMember `json:"members,omitempty"`
	Name string `json:"name,omitempty"`
	OrganizationId int32 `json:"organizationId,omitempty"`
	Uid string `json:"uid,omitempty"`
}
