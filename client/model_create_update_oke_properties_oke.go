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

type CreateUpdateOkePropertiesOke struct {
	Version   string                     `json:"version,omitempty"`
	NodePools map[string]NodePoolsOracle `json:"nodePools,omitempty"`
}
