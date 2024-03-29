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

type DeployedBackupBucketResponse struct {

	Id int32 `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	Cloud string `json:"cloud,omitempty"`

	SecretId string `json:"secretId,omitempty"`

	Status string `json:"status,omitempty"`

	InUse bool `json:"inUse,omitempty"`

	DeploymentId int32 `json:"deploymentId,omitempty"`

	ClusterId int32 `json:"clusterId,omitempty"`

	ClusterCloud string `json:"clusterCloud,omitempty"`

	ClusterDistribution string `json:"clusterDistribution,omitempty"`
}

// AssertDeployedBackupBucketResponseRequired checks if the required fields are not zero-ed
func AssertDeployedBackupBucketResponseRequired(obj DeployedBackupBucketResponse) error {
	return nil
}

// AssertRecurseDeployedBackupBucketResponseRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of DeployedBackupBucketResponse (e.g. [][]DeployedBackupBucketResponse), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseDeployedBackupBucketResponseRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aDeployedBackupBucketResponse, ok := obj.(DeployedBackupBucketResponse)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertDeployedBackupBucketResponseRequired(aDeployedBackupBucketResponse)
	})
}
