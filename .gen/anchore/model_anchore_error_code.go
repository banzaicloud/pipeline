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
// AnchoreErrorCode A description of an anchore error code (name, description)
type AnchoreErrorCode struct {
	// Error code name
	Name string `json:"name,omitempty"`
	// Description of the error code
	Description string `json:"description,omitempty"`
}
