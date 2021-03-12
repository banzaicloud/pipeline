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

// PersistDeploymentRequest describes an ARK deployment persisting request
type PersistDeploymentRequest struct {
	BucketID    uint
	RestoreMode bool
	Name        string
	Namespace   string
}

// EnableBackupServiceRequest describes an ARK service deployment request
type EnableBackupServiceRequest struct {
	CreateBucketRequest

	Schedule              string            `json:"schedule" binding:"required"`
	TTL                   string            `json:"ttl" binding:"required"`
	Labels                map[string]string `json:"labels,omitempty"`
	Options               BackupOptions     `json:"options,omitempty"`
	UseClusterSecret      bool              `json:"useClusterSecret,omitempty"`
	ServiceAccountRoleARN string            `json:"serviceAccountRoleARN,omitempty"`
	UseProviderSecret     bool              `json:"useProviderSecret,omitempty"`
}

// EnableBackupServiceResponse describes Pipeline's EnableBackupServiceRequest API response
type EnableBackupServiceResponse struct {
	Status int `json:"status"`
}

// DisableBackupServiceResponse describes Pipeline's DisableBackupServiceRequest API response
type DisableBackupServiceResponse struct {
	Status int `json:"status"`
}
