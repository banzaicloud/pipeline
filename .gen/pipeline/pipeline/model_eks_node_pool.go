/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments. 
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package pipeline

type EksNodePool struct {

	InstanceType string `json:"instanceType"`

	SpotPrice string `json:"spotPrice"`

	Autoscaling bool `json:"autoscaling,omitempty"`

	Count int32 `json:"count,omitempty"`

	MinCount int32 `json:"minCount"`

	MaxCount int32 `json:"maxCount"`

	Labels map[string]string `json:"labels,omitempty"`

	Image string `json:"image,omitempty"`

	Subnet EksSubnet `json:"subnet,omitempty"`
}
