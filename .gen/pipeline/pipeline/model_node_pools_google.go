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

type NodePoolsGoogle struct {

	Autoscaling bool `json:"autoscaling,omitempty"`

	Preemptible bool `json:"preemptible,omitempty"`

	Count int32 `json:"count"`

	MinCount int32 `json:"minCount,omitempty"`

	MaxCount int32 `json:"maxCount,omitempty"`

	InstanceType string `json:"instanceType"`

	Labels map[string]string `json:"labels,omitempty"`
}

// AssertNodePoolsGoogleRequired checks if the required fields are not zero-ed
func AssertNodePoolsGoogleRequired(obj NodePoolsGoogle) error {
	elements := map[string]interface{}{
		"count": obj.Count,
		"instanceType": obj.InstanceType,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	return nil
}

// AssertRecurseNodePoolsGoogleRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of NodePoolsGoogle (e.g. [][]NodePoolsGoogle), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseNodePoolsGoogleRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aNodePoolsGoogle, ok := obj.(NodePoolsGoogle)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertNodePoolsGoogleRequired(aNodePoolsGoogle)
	})
}
