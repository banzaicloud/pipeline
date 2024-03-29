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

// EksEncryptionConfigProvider - EKS encryption provider.
type EksEncryptionConfigProvider struct {

	// A customer master key to use for encryption. More information can be found at https://docs.aws.amazon.com/kms/latest/developerguide/create-keys.html.
	KeyARN string `json:"keyARN,omitempty"`
}

// AssertEksEncryptionConfigProviderRequired checks if the required fields are not zero-ed
func AssertEksEncryptionConfigProviderRequired(obj EksEncryptionConfigProvider) error {
	return nil
}

// AssertRecurseEksEncryptionConfigProviderRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of EksEncryptionConfigProvider (e.g. [][]EksEncryptionConfigProvider), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseEksEncryptionConfigProviderRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aEksEncryptionConfigProvider, ok := obj.(EksEncryptionConfigProvider)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertEksEncryptionConfigProviderRequired(aEksEncryptionConfigProvider)
	})
}
