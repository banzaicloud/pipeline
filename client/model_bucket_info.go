/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: 0.3.0
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

type BucketInfo struct {
	// the name of the object storage / bucket
	Name string `json:"name"`
	// true if the bucket has been created via piepeline
	Managed bool `json:"managed"`
	// true if the bucket has been created via piepeline
	Notes  string                `json:"notes,omitempty"`
	Secret BucketInfoSecret      `json:"secret,omitempty"`
	Azure  AzureBlobStorageProps `json:"azure,omitempty"`
	// the status of the bucket
	Status string `json:"status"`
}
