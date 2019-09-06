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

type CreatePkePropertiesNetwork struct {
	ApiServerAddress string `json:"apiServerAddress"`
	ServiceCIDR      string `json:"serviceCIDR"`
	PodCIDR          string `json:"podCIDR"`
	Provider         string `json:"provider"`
}
