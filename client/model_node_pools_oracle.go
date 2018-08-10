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

type NodePoolsOracle struct {
	Version string            `json:"version,omitempty"`
	Count   int32             `json:"count,omitempty"`
	Image   string            `json:"image,omitempty"`
	Shape   string            `json:"shape,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}
