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

type NodePoolsAkcs struct {
	Count              int32  `json:"count"`
	InstanceType       string `json:"instanceType"`
	SystemDiskSize     int32  `json:"systemDiskSize,omitempty"`
	SystemDiskCategory string `json:"systemDiskCategory,omitempty"`
}
