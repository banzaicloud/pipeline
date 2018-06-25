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

type SecretKeyValueAzure struct {
	AZURE_CLIENT_ID string `json:"AZURE_CLIENT_ID"`
	AZURE_CLIENT_SECRET string `json:"AZURE_CLIENT_SECRET"`
	AZURE_TENANT_ID string `json:"AZURE_TENANT_ID"`
	AZURE_SUBSCRIPTION_ID string `json:"AZURE_SUBSCRIPTION_ID"`
}
