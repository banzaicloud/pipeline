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

import (
	"time"
)

// CreateSecretResponse struct for CreateSecretResponse
type CreateSecretResponse struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Id        string    `json:"id"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	Version   int32     `json:"version,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}
