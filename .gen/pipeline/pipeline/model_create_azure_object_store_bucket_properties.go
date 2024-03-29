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

type CreateAzureObjectStoreBucketProperties struct {

	StorageAccount string `json:"storageAccount"`

	Location string `json:"location"`

	ResourceGroup string `json:"resourceGroup"`
}

// AssertCreateAzureObjectStoreBucketPropertiesRequired checks if the required fields are not zero-ed
func AssertCreateAzureObjectStoreBucketPropertiesRequired(obj CreateAzureObjectStoreBucketProperties) error {
	elements := map[string]interface{}{
		"storageAccount": obj.StorageAccount,
		"location": obj.Location,
		"resourceGroup": obj.ResourceGroup,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertRecurseCreateAzureObjectStoreBucketPropertiesRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of CreateAzureObjectStoreBucketProperties (e.g. [][]CreateAzureObjectStoreBucketProperties), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseCreateAzureObjectStoreBucketPropertiesRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aCreateAzureObjectStoreBucketProperties, ok := obj.(CreateAzureObjectStoreBucketProperties)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertCreateAzureObjectStoreBucketPropertiesRequired(aCreateAzureObjectStoreBucketProperties)
	})
}
