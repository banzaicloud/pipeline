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

type CreateAzurePropertiesAzure struct {
	ResourceGroup     string                    `json:"resourceGroup,omitempty"`
	KubernetesVersion string                    `json:"kubernetesVersion,omitempty"`
	NodePools         map[string]NodePoolsAzure `json:"nodePools,omitempty"`
}
