package storage // CreateBucketRequest describes a storage bucket creation

import (
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
)

// CreateBucketRequest to create bucket
type CreateBucketRequest struct {
	SecretId   string `json:"secret_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Properties struct {
		CreateAmazonObjectStoreBucketProperties *amazon.CreateAmazonObjectStoreBucketProperties `json:"amazon,omitempty"`
		CreateAzureObjectStoreBucketProperties  *azure.CreateAzureObjectStoreBucketProperties   `json:"azure,omitempty"`
		CreateGoogleObjectStoreBucketProperties *google.CreateGoogleObjectStoreBucketProperties `json:"google,omitempty"`
	} `json:"properties" binding:"required"`
}

// BucketInfo desribes a storage bucket
type BucketInfo struct {
	Name    string                          `json:"name"  binding:"required"`
	Managed bool                            `json:"managed" binding:"required"`
	Azure   *azure.BlobStoragePropsForAzure `json:"azure,omitempty"`
}

// CreateBucketResponse describes a storage bucket creation response
type CreateBucketResponse struct {
	Name string `json:"BucketName"`
}
