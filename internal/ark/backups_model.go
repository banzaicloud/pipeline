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

package ark

import (
	"encoding/json"
	"time"

	"emperror.dev/emperror"
	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/model"
)

// ClusterBackupsModel describes a cluster backup
type ClusterBackupsModel struct {
	ID uint `gorm:"primary_key"`

	UID            string
	Name           string
	Cloud          string
	Distribution   string
	NodeCount      uint
	ContentChecked bool
	StartedAt      *time.Time
	CompletedAt    *time.Time
	ExpireAt       *time.Time

	State []byte `sql:"type:json"`
	Nodes []byte `sql:"type:json"`

	Status        string
	StatusMessage string `sql:"type:text"`

	Organization   auth.Organization             `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint                          `gorm:"index;not null"`
	Cluster        model.ClusterModel            `gorm:"foreignkey:ClusterID"`
	ClusterID      uint                          `gorm:"index;not null"`
	Deployment     ClusterBackupDeploymentsModel `gorm:"foreignkey:DeploymentID"`
	DeploymentID   uint                          `gorm:"not null"`
	Bucket         ClusterBackupBucketsModel     `gorm:"foreignkey:BucketID"`
	BucketID       uint

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName changes the default table name
func (ClusterBackupsModel) TableName() string {
	return clusterBackupsTableName
}

// ConvertModelToEntity converts a ClusterBackupsModel to api.Backup
func (backup *ClusterBackupsModel) ConvertModelToEntity() *api.Backup {

	state := backup.GetStateObject()

	item := &api.Backup{
		ID:               backup.ID,
		UID:              backup.UID,
		Name:             backup.Name,
		TTL:              state.Spec.TTL,
		Labels:           state.ObjectMeta.Labels,
		Cloud:            backup.Cloud,
		Distribution:     backup.Distribution,
		Status:           backup.Status,
		StartAt:          state.Status.StartTimestamp.Time,
		ExpireAt:         state.Status.Expiration.Time,
		VolumeBackups:    state.Status.VolumeBackups,
		ValidationErrors: state.Status.ValidationErrors,
		ClusterID:        backup.ClusterID,
		ActiveClusterID:  backup.Bucket.Deployment.ClusterID,
		Options: api.BackupOptions{
			IncludedNamespaces:      state.Spec.IncludedNamespaces,
			IncludedResources:       state.Spec.IncludedResources,
			IncludeClusterResources: state.Spec.IncludeClusterResources,
			ExcludedNamespaces:      state.Spec.ExcludedNamespaces,
			ExcludedResources:       state.Spec.ExcludedResources,
			LabelSelector:           state.Spec.LabelSelector,
			SnapshotVolumes:         state.Spec.SnapshotVolumes,
		},
	}

	if backup.Bucket.ID > 0 {
		item.Bucket = backup.Bucket.ConvertModelToEntity()
	}

	return item
}

// GetStateObject gives back ark Backup from saved json
func (backup *ClusterBackupsModel) GetStateObject() *arkAPI.Backup {

	var stateObject *arkAPI.Backup
	err := json.Unmarshal(backup.State, &stateObject)
	if err != nil {
		return nil
	}

	return stateObject
}

// SetValuesFromRequest sets values from PersistBackupRequest to the model
func (backup *ClusterBackupsModel) SetValuesFromRequest(db *gorm.DB, req *api.PersistBackupRequest) error {

	var err error
	var stateJSON, nodesJSON []byte

	stateJSON, err = json.Marshal(req.Backup)
	if err != nil {
		return emperror.Wrap(err, "error converting backup to json")
	}

	if req.Nodes != nil {
		nodesJSON, err = json.Marshal(req.Nodes)
		if err != nil {
			return emperror.Wrap(err, "error converting nodes to json")
		}
	}

	backup.State = stateJSON
	// do not overwrite "Deleting" status with phase
	if backup.Status != "Deleting" {
		backup.Status = string(req.Backup.Status.Phase)
	}

	backup.Name = req.Backup.Name
	backup.UID = string(req.Backup.GetUID())

	// only update this in case of a new recordd
	if db.NewRecord(backup) {
		backup.Distribution = req.Distribution
		backup.Cloud = req.Cloud
		backup.DeploymentID = req.DeploymentID
		backup.ClusterID = req.ClusterID
	}

	// only update available node information once
	if backup.ContentChecked != true && req.Nodes != nil {
		backup.NodeCount = req.NodeCount
		backup.Nodes = nodesJSON
		backup.ContentChecked = req.ContentChecked
	}

	if !req.Backup.Status.StartTimestamp.IsZero() {
		backup.StartedAt = &req.Backup.Status.StartTimestamp.Time
	}
	if !req.Backup.Status.CompletionTimestamp.IsZero() {
		backup.CompletedAt = &req.Backup.Status.CompletionTimestamp.Time
	}
	if !req.Backup.Status.Expiration.IsZero() {
		backup.ExpireAt = &req.Backup.Status.Expiration.Time
	}

	return nil
}
