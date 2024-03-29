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

type EksNodePoolAllOf struct {

	Autoscaling NodePoolAutoScaling `json:"autoscaling,omitempty"`

	VolumeEncryption *EksNodePoolVolumeEncryption `json:"volumeEncryption,omitempty"`

	// Size of the EBS volume in GiBs of the nodes in the pool.
	VolumeSize int32 `json:"volumeSize,omitempty"`

	// Type of the EBS volume of the nodes in the pool (default gp3).
	VolumeType string `json:"volumeType,omitempty"`

	// Machine instance type.
	InstanceType string `json:"instanceType"`

	// Instance AMI.
	Image string `json:"image,omitempty"`

	// The upper limit price for the requested spot instance. If this field is left empty or 0 passed in on-demand instances used instead of spot instances.
	SpotPrice string `json:"spotPrice,omitempty"`

	SubnetId string `json:"subnetId,omitempty"`

	// List of additional custom security groups for all nodes in the pool.
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// Setup available instance stores (NVMe disks) to use for Kubelet root if available. As a result emptyDir volumes will be provisioned on local instance storage disks. You can check out available instance storages here https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-volumes.
	UseInstanceStore bool `json:"useInstanceStore,omitempty"`
}

// AssertEksNodePoolAllOfRequired checks if the required fields are not zero-ed
func AssertEksNodePoolAllOfRequired(obj EksNodePoolAllOf) error {
	elements := map[string]interface{}{
		"instanceType": obj.InstanceType,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	if err := AssertNodePoolAutoScalingRequired(obj.Autoscaling); err != nil {
		return err
	}
	if obj.VolumeEncryption != nil {
		if err := AssertEksNodePoolVolumeEncryptionRequired(*obj.VolumeEncryption); err != nil {
			return err
		}
	}
	return nil
}

// AssertRecurseEksNodePoolAllOfRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of EksNodePoolAllOf (e.g. [][]EksNodePoolAllOf), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseEksNodePoolAllOfRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aEksNodePoolAllOf, ok := obj.(EksNodePoolAllOf)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertEksNodePoolAllOfRequired(aEksNodePoolAllOf)
	})
}
