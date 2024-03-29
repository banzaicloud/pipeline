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

type PkeAwsUpdateNodePoolRequestAllOf struct {

	Autoscaling NodePoolAutoScaling `json:"autoscaling,omitempty"`

	// Size of the EBS volume in GiBs of the nodes in the pool.
	VolumeSize int32 `json:"volumeSize,omitempty"`

	// The instance type to use for your node pool.
	InstanceType string `json:"instanceType,omitempty"`

	// The instance AMI to use for your node pool.
	Image string `json:"image,omitempty"`

	// The upper limit price for the requested spot instance. If this field is empty or 0 on-demand instances are used instead of spot instances.
	SpotPrice string `json:"spotPrice,omitempty"`

	Options BaseUpdateNodePoolOptions `json:"options,omitempty"`
}

// AssertPkeAwsUpdateNodePoolRequestAllOfRequired checks if the required fields are not zero-ed
func AssertPkeAwsUpdateNodePoolRequestAllOfRequired(obj PkeAwsUpdateNodePoolRequestAllOf) error {
	if err := AssertNodePoolAutoScalingRequired(obj.Autoscaling); err != nil {
		return err
	}
	if err := AssertBaseUpdateNodePoolOptionsRequired(obj.Options); err != nil {
		return err
	}
	return nil
}

// AssertRecursePkeAwsUpdateNodePoolRequestAllOfRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of PkeAwsUpdateNodePoolRequestAllOf (e.g. [][]PkeAwsUpdateNodePoolRequestAllOf), otherwise ErrTypeAssertionError is thrown.
func AssertRecursePkeAwsUpdateNodePoolRequestAllOfRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aPkeAwsUpdateNodePoolRequestAllOf, ok := obj.(PkeAwsUpdateNodePoolRequestAllOf)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertPkeAwsUpdateNodePoolRequestAllOfRequired(aPkeAwsUpdateNodePoolRequestAllOf)
	})
}
