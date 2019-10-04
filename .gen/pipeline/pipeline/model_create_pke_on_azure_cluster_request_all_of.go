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

type CreatePkeOnAzureClusterRequestAllOf struct {

	// Non-existent resources will be created in this location. Existing resources that must have the same location as the cluster will be validated against this.
	Location string `json:"location,omitempty"`

	// Required resources will be created in this resource group.
	ResourceGroup string `json:"resourceGroup"`

	Network PkeOnAzureClusterNetwork `json:"network,omitempty"`

	Nodepools []PkeOnAzureNodePool `json:"nodepools,omitempty"`
}
