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

type BackupOptions struct {
	IncludedNamespaces      []string `json:"includedNamespaces,omitempty"`
	IncludedResources       []string `json:"includedResources,omitempty"`
	ExcludedNamespaces      []string `json:"excludedNamespaces,omitempty"`
	ExcludedResources       []string `json:"excludedResources,omitempty"`
	SnapshotVolumes         bool     `json:"snapshotVolumes,omitempty"`
	IncludeClusterResources bool     `json:"includeClusterResources,omitempty"`
}
