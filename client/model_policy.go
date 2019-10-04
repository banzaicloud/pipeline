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
// Policy struct for Policy
type Policy struct {
	Id string `json:"id"`
	Name string `json:"name,omitempty"`
	Comment string `json:"comment,omitempty"`
	Version string `json:"version"`
	Rules []PolicyRule `json:"rules,omitempty"`
}
