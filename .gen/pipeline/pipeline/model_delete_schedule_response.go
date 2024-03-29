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

type DeleteScheduleResponse struct {

	Name string `json:"name,omitempty"`

	Status int32 `json:"status,omitempty"`
}

// AssertDeleteScheduleResponseRequired checks if the required fields are not zero-ed
func AssertDeleteScheduleResponseRequired(obj DeleteScheduleResponse) error {
	return nil
}

// AssertRecurseDeleteScheduleResponseRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of DeleteScheduleResponse (e.g. [][]DeleteScheduleResponse), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseDeleteScheduleResponseRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aDeleteScheduleResponse, ok := obj.(DeleteScheduleResponse)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertDeleteScheduleResponseRequired(aDeleteScheduleResponse)
	})
}
