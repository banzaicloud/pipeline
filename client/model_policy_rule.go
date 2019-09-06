/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

// A rule that defines and decision value if the match is found true for a given image.
type PolicyRule struct {
	Id      string             `json:"id,omitempty"`
	Gate    string             `json:"gate"`
	Trigger string             `json:"trigger"`
	Action  string             `json:"action"`
	Params  []PolicyRuleParams `json:"params,omitempty"`
}
