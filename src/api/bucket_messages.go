// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

// CreateBucketRequest to create bucket
type CreateBucketRequest struct {
	SecretId   string `json:"secretId"`
	SecretName string `json:"secretName"`
	Name       string `json:"name" binding:"required"`
	Properties struct {
		Amazon *CreateAmazonObjectStoreBucketProperties `json:"amazon,omitempty"`
		Azure  *CreateAzureObjectStoreBucketProperties  `json:"azure,omitempty"`
		Google *CreateGoogleObjectStoreBucketProperties `json:"google,omitempty"`
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

// CreateBucketResponse describes a storage bucket creation response
type CreateBucketResponse struct {
	BucketName string `json:"name"`
	CloudType  string `json:"cloud"`
}
