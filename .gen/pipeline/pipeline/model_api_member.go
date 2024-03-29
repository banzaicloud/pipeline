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

type ApiMember struct {

	Cloud string `json:"cloud,omitempty"`

	Distribution string `json:"distribution,omitempty"`

	Id int32 `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	Status string `json:"status,omitempty"`
}

// AssertApiMemberRequired checks if the required fields are not zero-ed
func AssertApiMemberRequired(obj ApiMember) error {
	return nil
}

// AssertRecurseApiMemberRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of ApiMember (e.g. [][]ApiMember), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseApiMemberRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aApiMember, ok := obj.(ApiMember)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertApiMemberRequired(aApiMember)
	})
}
