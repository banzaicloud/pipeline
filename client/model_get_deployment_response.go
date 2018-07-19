/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: 0.3.0
 * Contact: info@banzaicloud.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package client

type GetDeploymentResponse struct {
	ReleaseName string `json:"releaseName,omitempty"`
	Chart       string `json:"chart,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Version     int32  `json:"version,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	Status      string `json:"status,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	// deployment notes in base64 encoded format
	Notes string `json:"notes,omitempty"`
	// deployment values in base64 encoded YAML format
	Values string `json:"values,omitempty"`
}
