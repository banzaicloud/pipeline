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

type EnableArkRequest struct {

	Cloud string `json:"cloud"`

	BucketName string `json:"bucketName"`

	Schedule string `json:"schedule"`

	Ttl string `json:"ttl"`

	SecretId string `json:"secretId"`

	Location string `json:"location,omitempty"`

	// relevant only in case of Amazon clusters. By default set to false in which case you must add snapshot permissions to your node instance role. Should you set to true Pipeline will deploy your cluster secret to the cluster.
	UseClusterSecret bool `json:"useClusterSecret,omitempty"`

	// relevant only in case of Amazon clusters. This a third option to give permissions for volume snapshots to Velero, besides the default NodeInstance role or cluster secret deployment.
	ServiceAccountRoleARN string `json:"serviceAccountRoleARN,omitempty"`

	// required only case of Azure
	StorageAccount string `json:"storageAccount,omitempty"`

	// required only case of Azure
	ResourceGroup string `json:"resourceGroup,omitempty"`

	Labels Labels `json:"labels,omitempty"`

	Options BackupOptions `json:"options,omitempty"`
}
