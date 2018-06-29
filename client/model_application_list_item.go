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

type ApplicationListItem struct {
	Id            int32  `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
	ClusterName   string `json:"clusterName,omitempty"`
	ClusterId     int32  `json:"clusterId,omitempty"`
	Status        string `json:"status,omitempty"`
	CatalogName   string `json:"catalogName,omitempty"`
	Icon          string `json:"icon,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}
