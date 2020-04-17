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

type CreatePkeOnVsphereClusterRequestAllOf struct {

	// Secret ID used to setup VSphere storage classes. Overrides the default settings in main cluster secret.
	StorageSecretId string `json:"storageSecretId,omitempty"`

	// Secret name used to setup VSphere storage classes. Overrides default value from the main cluster secret.
	StorageSecretName string `json:"storageSecretName,omitempty"`

	// Folder to create nodes in. Overrides default value from the main cluster secret.
	Folder string `json:"folder,omitempty"`

	// Name of datastore or datastore cluster to place VM disks on. Overrides default value from the main cluster secret.
	Datastore string `json:"datastore,omitempty"`

	// Virtual machines will be created in this resource pool. Overrides default value from the main cluster secret.
	ResourcePool string `json:"resourcePool,omitempty"`

	NodePools []PkeOnVsphereNodePool `json:"nodePools,omitempty"`

	// IPv4 range to allocate addresses for LoadBalancer Services (MetalLB)
	LoadBalancerIpRange string `json:"loadBalancerIpRange,omitempty"`
}
