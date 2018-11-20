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

type LaunchSpotguidesRequest struct {
	RepoOrganization string                 `json:"repoOrganization"`
	RepoName         string                 `json:"repoName"`
	RepoPrivate      bool                   `json:"repoPrivate,omitempty"`
	RepoLatent       bool                   `json:"repoLatent,omitempty"`
	SpotguideName    string                 `json:"spotguideName"`
	SpotguideVersion string                 `json:"spotguideVersion,omitempty"`
	Cluster          CreateClusterRequest   `json:"cluster,omitempty"`
	Secrets          []CreateSecretRequest  `json:"secrets,omitempty"`
	Pipeline         map[string]interface{} `json:"pipeline,omitempty"`
}
