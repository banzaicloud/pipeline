/*
 * Anchore Engine API Server
 *
 * This is the Anchore Engine API. Provides the primary external API for users of the service.
 *
 * API version: 0.1.12
 * Contact: nurmi@anchore.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package anchore
// TriggerParamSpec struct for TriggerParamSpec
type TriggerParamSpec struct {
	// Parameter name as it appears in policy document
	Name string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// An example value for the parameter (encoded as a string if the parameter is an object or list type)
	Example *string `json:"example,omitempty"`
	// State of the trigger parameter
	State string `json:"state,omitempty"`
	// The name of another trigger that supercedes this on functionally if this is deprecated
	SupercededBy *string `json:"superceded_by,omitempty"`
	// Is this a required parameter or optional
	Required bool `json:"required,omitempty"`
	// If present, a definition for validation of input. Typically a jsonschema object that can be used to validate an input against.
	Validator map[string]interface{} `json:"validator,omitempty"`
}
