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

// PkeAwsUpdateNodePoolRequest - Node pool update request object for an PKE AWS cluster.
type PkeAwsUpdateNodePoolRequest struct {

	// Node pool size.
	Size int32 `json:"size"`

	// Node pool labels.
	Labels map[string]string `json:"labels,omitempty"`

	Autoscaling NodePoolAutoScaling `json:"autoscaling,omitempty"`

	// Size of the EBS volume in GBs of the nodes in the pool.
	VolumeSize int32 `json:"volumeSize,omitempty"`

	// The instance type to use for your node pool.
	InstanceType string `json:"instanceType,omitempty"`

	// The instance AMI to use for your node pool.
	Image string `json:"image,omitempty"`

	// The upper limit price for the requested spot instance. If this field is empty or 0 on-demand instances are used instead of spot instances.
	SpotPrice string `json:"spotPrice,omitempty"`

	Options BaseUpdateNodePoolOptions `json:"options,omitempty"`
}
