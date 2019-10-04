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
// UserCreationRequest A payload for creating a new user, includes the username and password in a single request
type UserCreationRequest struct {
	// The username to create
	Username string `json:"username"`
	// The initial password for the user, must be at least 6 characters, up to 128
	Password string `json:"password"`
}
