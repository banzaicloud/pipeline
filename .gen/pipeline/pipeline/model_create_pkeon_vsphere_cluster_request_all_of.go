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

type CreatePkeonVsphereClusterRequestAllOf struct {

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

	Nodepools []PkeonVsphereNodePool `json:"nodepools,omitempty"`

	// IPv4 range to allocate addresses for LoadBalancer Services (MetalLB)
	LoadBalancerIPRange string `json:"loadBalancerIPRange,omitempty"`
}

// AssertCreatePkeonVsphereClusterRequestAllOfRequired checks if the required fields are not zero-ed
func AssertCreatePkeonVsphereClusterRequestAllOfRequired(obj CreatePkeonVsphereClusterRequestAllOf) error {
	for _, el := range obj.Nodepools {
		if err := AssertPkeonVsphereNodePoolRequired(el); err != nil {
			return err
		}
	}
	return nil
}

// AssertRecurseCreatePkeonVsphereClusterRequestAllOfRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of CreatePkeonVsphereClusterRequestAllOf (e.g. [][]CreatePkeonVsphereClusterRequestAllOf), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseCreatePkeonVsphereClusterRequestAllOfRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aCreatePkeonVsphereClusterRequestAllOf, ok := obj.(CreatePkeonVsphereClusterRequestAllOf)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertCreatePkeonVsphereClusterRequestAllOfRequired(aCreatePkeonVsphereClusterRequestAllOf)
	})
}
