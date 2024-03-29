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

type PkeonAzureClusterNetwork struct {

	Name string `json:"name,omitempty"`

	// When referencing an already existing virtual network this field does not need to be specified.
	Cidr string `json:"cidr,omitempty"`
}

// AssertPkeonAzureClusterNetworkRequired checks if the required fields are not zero-ed
func AssertPkeonAzureClusterNetworkRequired(obj PkeonAzureClusterNetwork) error {
	return nil
}

// AssertRecursePkeonAzureClusterNetworkRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of PkeonAzureClusterNetwork (e.g. [][]PkeonAzureClusterNetwork), otherwise ErrTypeAssertionError is thrown.
func AssertRecursePkeonAzureClusterNetworkRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aPkeonAzureClusterNetwork, ok := obj.(PkeonAzureClusterNetwork)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertPkeonAzureClusterNetworkRequired(aPkeonAzureClusterNetwork)
	})
}
