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

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// CreateBucketRequest describes create bucket request
type CreateBucketRequest struct {
	Cloud      string `json:"cloud" binding:"required"`
	BucketName string `json:"bucketName" binding:"required"`
	SecretID   string `json:"secretId" binding:"required"`
	Location   string `json:"location"`

	AzureBucketProperties `json:"azure"`
}

// AzureObjectStoreBucketProperties describes bucket properties for an Azure ObjectStore Container
type AzureBucketProperties struct {
	StorageAccount string `json:"storageAccount,omitempty"`
	ResourceGroup  string `json:"resourceGroup,omitempty"`
}

// FindBucketRequest describes a find bucket request
type FindBucketRequest struct {
	Cloud      string
	BucketName string
	Location   string
}

// Bucket describes a Bucket used for ARK backups
type Bucket struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Cloud    string `json:"cloud"`
	SecretID string `json:"secretId"`
	Location string `json:"location,omitempty"`
	AzureBucketProperties
	Status              string                    `json:"status"`
	InUse               bool                      `json:"inUse"`
	DeploymentID        uint                      `json:"deploymentId,omitempty"`
	ClusterID           uint                      `json:"clusterId,omitempty"`
	ClusterCloud        string                    `json:"clusterCloud,omitempty"`
	ClusterDistribution pkgCluster.DistributionID `json:"clusterDistribution,omitempty"`
}

// DeleteBucketResponse describes a delete bucket response
type DeleteBucketResponse struct {
	ID     uint `json:"id"`
	Status int  `json:"status"`
}
