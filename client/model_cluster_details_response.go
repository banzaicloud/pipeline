/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: 0.3.0
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

type ClusterDetailsResponse struct {
	Status        string                    `json:"status,omitempty"`
	StatusMessage string                    `json:"statusMessage,omitempty"`
	Name          string                    `json:"name,omitempty"`
	Cloud         string                    `json:"cloud,omitempty"`
	Distribution  string                    `json:"distribution,omitempty"`
	Spot          bool                      `json:"spot,omitempty"`
	Location      string                    `json:"location,omitempty"`
	Id            int32                     `json:"id,omitempty"`
	Logging       bool                      `json:"logging,omitempty"`
	Monitoring    bool                      `json:"monitoring,omitempty"`
	Securityscan  bool                      `json:"securityscan,omitempty"`
	CreatedAt     string                    `json:"createdAt,omitempty"`
	CreatorName   string                    `json:"creatorName,omitempty"`
	CreatorId     int32                     `json:"creatorId,omitempty"`
	Region        string                    `json:"region,omitempty"`
	NodePools     map[string]NodePoolStatus `json:"nodePools,omitempty"`
	SecretId      string                    `json:"secretId,omitempty"`
	SecretName    string                    `json:"secretName,omitempty"`
	Endpoint      string                    `json:"endpoint,omitempty"`
	Master        ResourceSummaryItem       `json:"master,omitempty"`
	TotalSummary  PodItemResourceSummary    `json:"totalSummary,omitempty"`
}
