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

type BucketInfoSecret struct {

	Id string `json:"id"`

	Name string `json:"name,omitempty"`

	// the secret identifier of the azure access information
	AccessId string `json:"accessId,omitempty"`

	// the secret name of the azure access information
	AccessName string `json:"accessName,omitempty"`
}

// AssertBucketInfoSecretRequired checks if the required fields are not zero-ed
func AssertBucketInfoSecretRequired(obj BucketInfoSecret) error {
	elements := map[string]interface{}{
		"id": obj.Id,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertRecurseBucketInfoSecretRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of BucketInfoSecret (e.g. [][]BucketInfoSecret), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseBucketInfoSecretRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aBucketInfoSecret, ok := obj.(BucketInfoSecret)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertBucketInfoSecretRequired(aBucketInfoSecret)
	})
}
