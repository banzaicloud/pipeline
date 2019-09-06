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

type DeploymentScaleStatus struct {
	CurrentReplicas int32  `json:"currentReplicas,omitempty"`
	DesiredReplicas int32  `json:"desiredReplicas,omitempty"`
	Message         string `json:"message,omitempty"`
}
