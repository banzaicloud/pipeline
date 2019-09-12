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

type CreateClusterRequest struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Cloud    string `json:"cloud"`
	// The lifespan of the cluster expressed in minutes after which it is automatically deleted. Zero value means the cluster is never automatically deleted.
	TtlMinutes   int32                  `json:"ttlMinutes,omitempty"`
	SecretId     string                 `json:"secretId,omitempty"`
	SecretIds    []string               `json:"secretIds,omitempty"`
	SecretName   string                 `json:"secretName,omitempty"`
	PostHooks    map[string]interface{} `json:"postHooks,omitempty"`
	ScaleOptions ScaleOptions           `json:"scaleOptions,omitempty"`
	Properties   map[string]interface{} `json:"properties"`
}
