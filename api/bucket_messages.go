package api

// CreateBucketRequest to create bucket
type CreateBucketRequest struct {
	SecretId   string `json:"secret_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Properties struct {
		Amazon *CreateAmazonObjectStoreBucketProperties `json:"amazon,omitempty"`
		Azure  *CreateAzureObjectStoreBucketProperties  `json:"azure,omitempty"`
		Google *CreateGoogleObjectStoreBucketProperties `json:"google,omitempty"`
		Oracle *CreateObjectStoreBucketProperties       `json:"oracle,omitempty"`
	} `json:"properties" binding:"required"`
}

// CreateAmazonObjectStoreBucketProperties describes the properties of
// S3 bucket creation request
type CreateAmazonObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}

// CreateAzureObjectStoreBucketProperties describes an Azure ObjectStore Container Creation request
type CreateAzureObjectStoreBucketProperties struct {
	Location       string `json:"location" binding:"required"`
	StorageAccount string `json:"storageAccount"`
	ResourceGroup  string `json:"resourceGroup"`
}

// CreateGoogleObjectStoreBucketProperties describes Google Object Store Bucket creation request
type CreateGoogleObjectStoreBucketProperties struct {
	Location string `json:"location,required"`
}

// CreateObjectStoreBucketProperties describes Oracle Object Store Bucket creation request
type CreateObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}

// CreateBucketResponse describes a storage bucket creation response
type CreateBucketResponse struct {
	Name string `json:"BucketName"`
}
