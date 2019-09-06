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

type DeployedBackupBucketResponse struct {
	Id                  int32  `json:"id,omitempty"`
	Name                string `json:"name,omitempty"`
	Cloud               string `json:"cloud,omitempty"`
	SecretId            string `json:"secretId,omitempty"`
	Status              string `json:"status,omitempty"`
	InUse               bool   `json:"inUse,omitempty"`
	DeploymentId        int32  `json:"deploymentId,omitempty"`
	ClusterId           int32  `json:"clusterId,omitempty"`
	ClusterCloud        string `json:"clusterCloud,omitempty"`
	ClusterDistribution string `json:"clusterDistribution,omitempty"`
}
