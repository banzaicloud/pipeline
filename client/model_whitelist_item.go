/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments. 
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client
// WhitelistItem Identifies a specific gate and trigger match from a policy against an image and indicates it should be ignored in final policy decisions
type WhitelistItem struct {
	Id string `json:"id,omitempty"`
	Gate string `json:"gate"`
	TriggerId string `json:"trigger_id"`
}
