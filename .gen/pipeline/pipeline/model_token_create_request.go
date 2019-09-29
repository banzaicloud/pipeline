/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments. 
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package pipeline

import (
	"time"
)

type TokenCreateRequest struct {

	Name string `json:"name,omitempty"`

	VirtualUser string `json:"virtualUser,omitempty"`

	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
