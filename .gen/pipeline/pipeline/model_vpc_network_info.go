/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments. 
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package pipeline

type VpcNetworkInfo struct {

	// The IPv4 CIDR blocks assigned to the VPC network
	Cidrs []string `json:"cidrs"`

	// Identifier of the VPC network
	Id string `json:"id"`

	// Name of the VPC network
	Name string `json:"name,omitempty"`
}
