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

type UpdateNodePoolRequest struct {

	// Node pool size.
	Size int32 `json:"size"`

	// Node pool labels.
	Labels map[string]string `json:"labels,omitempty"`

	Autoscaling NodePoolAutoScaling `json:"autoscaling,omitempty"`

	VolumeEncryption *EksNodePoolVolumeEncryption `json:"volumeEncryption,omitempty"`

	// Size of the EBS volume in GiBs of the nodes in the pool.
	VolumeSize int32 `json:"volumeSize,omitempty"`

	// Type of the EBS volume of the nodes in the pool (default gp3).
	VolumeType string `json:"volumeType,omitempty"`

	// The instance type to use for your node pool.
	InstanceType string `json:"instanceType,omitempty"`

	// The instance AMI to use for your node pool.
	Image string `json:"image,omitempty"`

	// The Kubernetes version to use for your node pool.
	Version string `json:"version,omitempty"`

	// The upper limit price for the requested spot instance. If this field is empty or 0 on-demand instances are used instead of spot instances.
	SpotPrice string `json:"spotPrice,omitempty"`

	// List of additional custom security groups for all nodes in the pool.
	SecurityGroups *[]string `json:"securityGroups,omitempty"`

	// Setup available instance stores (NVMe disks) to use for Kubelet root if available. As a result emptyDir volumes will be provisioned on local instance storage disks. You can check out available instance storages here https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-volumes.
	UseInstanceStore *bool `json:"useInstanceStore,omitempty"`

	Options BaseUpdateNodePoolOptions `json:"options,omitempty"`
}
