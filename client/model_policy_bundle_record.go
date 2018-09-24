/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: 0.3.0
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"time"
)

// A policy bundle plus some metadata
type PolicyBundleRecord struct {
	CreatedAt   time.Time `json:"created_at,omitempty"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
	// The bundle's identifier
	PolicyId string `json:"policyId,omitempty"`
	// True if the bundle is currently defined to be used automatically
	Active bool `json:"active,omitempty"`
	// UserId of the user that owns the bundle
	UserId string `json:"userId,omitempty"`
	// Source location of where the policy bundle originated
	PolicySource string       `json:"policy_source,omitempty"`
	Policybundle PolicyBundle `json:"policybundle,omitempty"`
}
