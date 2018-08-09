package storage // CreateBucketRequest describes a storage bucket creation

import (
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/objectstore"
)

// CreateBucketRequest to create bucket
type CreateBucketRequest struct {
	SecretId   string `json:"secret_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Properties struct {
		CreateAmazonObjectStoreBucketProperties *ec2.CreateAmazonObjectStoreBucketProperties `json:"ec2,omitempty"`
		CreateAzureObjectStoreBucketProperties  *aks.CreateAzureObjectStoreBucketProperties  `json:"aks,omitempty"`
		CreateGoogleObjectStoreBucketProperties *gke.CreateGoogleObjectStoreBucketProperties `json:"gke,omitempty"`
		CreateOracleObjectStoreBucketProperties *oracle.CreateObjectStoreBucketProperties    `json:"oracle,omitempty"`
	} `json:"properties" binding:"required"`
}

// BucketInfo desribes a storage bucket
type BucketInfo struct {
	Name     string                        `json:"name"  binding:"required"`
	Managed  bool                          `json:"managed" binding:"required"`
	Location string                        `json:"location,omitempty"`
	Azure    *aks.BlobStoragePropsForAzure `json:"aks,omitempty"`
}

// CreateBucketResponse describes a storage bucket creation response
type CreateBucketResponse struct {
	Name string `json:"BucketName"`
}
