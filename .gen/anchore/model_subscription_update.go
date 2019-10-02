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

// A modification to a subscription entry to change its status or value
type SubscriptionUpdate struct {
	// The new subscription value, e.g. the new tag to be subscribed to
	SubscriptionValue *string `json:"subscription_value,omitempty"`
	// Toggle the subscription processing on or off
	Active bool `json:"active,omitempty"`
}
