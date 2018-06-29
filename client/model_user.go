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

type User struct {
	Id            int32                  `json:"id,omitempty"`
	CreatedAt     string                 `json:"createdAt,omitempty"`
	UpdatedAt     string                 `json:"updatedAt,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Email         string                 `json:"email,omitempty"`
	Login         string                 `json:"login,omitempty"`
	Image         string                 `json:"image,omitempty"`
	Organizations map[string]interface{} `json:"organizations,omitempty"`
}
