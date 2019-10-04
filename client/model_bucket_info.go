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

// BucketInfo struct for BucketInfo
type BucketInfo struct {
	// the name of the object storage / bucket
	Name string `json:"name"`
	// true if the bucket has been created via pipeline
	Managed bool `json:"managed"`
	// cloud provider of the bucket
	Cloud string `json:"cloud"`
	// location of the bucket
	Location string `json:"location"`
	// notes for the bucket
	Notes  string                `json:"notes,omitempty"`
	Secret BucketInfoSecret      `json:"secret,omitempty"`
	Aks    AzureBlobStorageProps `json:"aks,omitempty"`
	Oracle OracleStorageProps    `json:"oracle,omitempty"`
	// the status of the bucket
	Status string `json:"status"`
	// the reason for the error status
	StatusMessage string `json:"statusMessage,omitempty"`
}
