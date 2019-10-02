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
import (
	"time"
)

// A username for authenticating with one or more types of credentials. User type defines the expected credentials allowed for the user. Native users have passwords, External users have no credential internally. Internal users are service/system users for inter-service communication.
type User struct {
	// The username to authenticate with
	Username string `json:"username"`
	// The user's type
	Type string `json:"type,omitempty"`
	// If the user is external, this is the source that the user was initialized from. All other user types have this set to null
	Source string `json:"source,omitempty"`
	// The timestampt the user record was created
	CreatedAt time.Time `json:"created_at,omitempty"`
	// The timestamp of the last update to this record
	LastUpdated time.Time `json:"last_updated,omitempty"`
}
