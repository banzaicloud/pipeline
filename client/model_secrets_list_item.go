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

type SecretsListItem struct {
	Id string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Version int32 `json:"version,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Values map[string]interface{} `json:"values,omitempty"`
}
