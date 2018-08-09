package api

import (
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
		Azure                                   *CreateAzureObjectStoreBucketProperties      `json:"aks,omitempty"`
		CreateGoogleObjectStoreBucketProperties *gke.CreateGoogleObjectStoreBucketProperties `json:"gke,omitempty"`
		CreateOracleObjectStoreBucketProperties *oracle.CreateObjectStoreBucketProperties    `json:"oracle,omitempty"`
	} `json:"properties" binding:"required"`
}

// CreateAzureObjectStoreBucketProperties describes an Azure ObjectStore Container Creation request
type CreateAzureObjectStoreBucketProperties struct {
	Location       string `json:"location" binding:"required"`
	StorageAccount string `json:"storageAccount"`
	ResourceGroup  string `json:"resourceGroup"`
}

// CreateBucketResponse describes a storage bucket creation response
type CreateBucketResponse struct {
	Name string `json:"BucketName"`
}
