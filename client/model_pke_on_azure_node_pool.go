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

// PkeOnAzureNodePool struct for PkeOnAzureNodePool
type PkeOnAzureNodePool struct {
	Name         string                   `json:"name"`
	Roles        []string                 `json:"roles"`
	Labels       map[string]string        `json:"labels,omitempty"`
	Subnet       PkeOnAzureNodePoolSubnet `json:"subnet,omitempty"`
	Zones        []string                 `json:"zones,omitempty"`
	Autoscaling  bool                     `json:"autoscaling,omitempty"`
	MinCount     int32                    `json:"minCount,omitempty"`
	MaxCount     int32                    `json:"maxCount,omitempty"`
	Count        int32                    `json:"count,omitempty"`
	InstanceType string                   `json:"instanceType"`
}
