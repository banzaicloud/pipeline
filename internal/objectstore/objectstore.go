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

package objectstore

// ObjectStoreService is the interface that cloud specific object store implementation
// must implement
type ObjectStoreService interface {
	CreateBucket(string) error
	ListBuckets() ([]*BucketInfo, error)
	ListManagedBuckets() ([]*BucketInfo, error)
	DeleteBucket(string) error
	CheckBucket(string) error
}

// BucketInfo describes a storage bucket
type BucketInfo struct {
	Name            string                    `json:"name"  binding:"required"`
	Managed         bool                      `json:"managed" binding:"required"`
	Location        string                    `json:"location,omitempty"`
	SecretRef       string                    `json:"secretId,omitempty"`
	Cloud           string                    `json:"cloud,omitempty"`
	Azure           *BlobStoragePropsForAzure `json:"aks,omitempty"`
	Status          string                    `json:"status"`
	StatusMsg       string                    `json:"statusMsg"`
	AccessSecretRef string                    `json:"accessSecretId"`
}

// BlobStoragePropsForAzure describes the Azure specific properties
type BlobStoragePropsForAzure struct {
	ResourceGroup  string `json:"resourceGroup" binding:"required"`
	StorageAccount string `json:"storageAccount" binding:"required"`
}
