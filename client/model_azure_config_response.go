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

type AzureConfigResponse struct {
	Type string `json:"type,omitempty"`
	NameRegexp string `json:"nameRegexp,omitempty"`
	Locations string `json:"locations,omitempty"`
	InstanceType AzureConfigResponseInstanceType `json:"instanceType,omitempty"`
	KubernetesVersions []string `json:"kubernetes_versions,omitempty"`
}
