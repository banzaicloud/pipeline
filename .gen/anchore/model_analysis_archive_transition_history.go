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

// AnalysisArchiveTransitionHistory A rule for auto-archiving image analysis by time and/or tag-history
type AnalysisArchiveTransitionHistory struct {
	// The task that created & updated this entry
	TransitionTaskId string    `json:"transition_task_id,omitempty"`
	RuleId           string    `json:"rule_id,omitempty"`
	ImageDigest      string    `json:"imageDigest,omitempty"`
	Transition       string    `json:"transition,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	LastUpdated      time.Time `json:"last_updated,omitempty"`
}
