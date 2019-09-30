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

type NodePoolsAzure struct {

	Autoscaling bool `json:"autoscaling,omitempty"`

	Count int32 `json:"count"`

	MinCount int32 `json:"minCount,omitempty"`

	MaxCount int32 `json:"maxCount,omitempty"`

	InstanceType string `json:"instanceType"`

	Labels map[string]string `json:"labels,omitempty"`
}
