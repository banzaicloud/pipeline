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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/model"
)

// ClusterBackupRestoresModel describes an ARK restore model
type ClusterBackupRestoresModel struct {
	ID uint `gorm:"primary_key"`

	UID  string
	Name string

	State    []byte `sql:"type:json"`
	Results  []byte `sql:"type:json"`
	Warnings uint
	Errors   uint

	Bucket         ClusterBackupBucketsModel `gorm:"foreignkey:BucketID"`
	BucketID       uint                      `gorm:"index;not null"`
	Cluster        model.ClusterModel        `gorm:"foreignkey:ClusterID"`
	ClusterID      uint                      `gorm:"index;not null"`
	Organization   auth.Organization         `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint                      `gorm:"index;not null"`

	Status        string
	StatusMessage string `sql:"type:text;"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// TableName changes the default table name
func (ClusterBackupRestoresModel) TableName() string {
	return clusterBackupRestoresTableName
}

// SetValuesFromRequest set values from a PersistRestoreRequest to the restore object
func (restore *ClusterBackupRestoresModel) SetValuesFromRequest(req *api.PersistRestoreRequest) error {

	stateJSON, err := json.Marshal(req.Restore)
	if err != nil {
		return emperror.Wrap(err, "error converting state to json")
	}

	resultsJSON, err := json.Marshal(req.Results)
	if err != nil {
		return emperror.Wrap(err, "error converting results to json")
	}

	restore.Results = resultsJSON
	restore.State = stateJSON
	// do not overwrite "Deleting" status with phase
	if restore.Status != "Deleting" {
		restore.Status = string(req.Restore.Status.Phase)
	}
	restore.Name = req.Restore.Name
	restore.Warnings = uint(req.Restore.Status.Warnings)
	restore.Errors = uint(req.Restore.Status.Errors)

	return nil
}

// ConvertModelToEntity converts ClusterBackupRestoresModel to api.Restore
func (restore *ClusterBackupRestoresModel) ConvertModelToEntity() *api.Restore {

	state := restore.GetState()
	results := restore.GetResults()

	item := &api.Restore{
		ID:               restore.ID,
		UID:              restore.UID,
		Name:             restore.Name,
		BackupName:       state.Spec.BackupName,
		Status:           restore.Status,
		Warnings:         restore.Warnings,
		Errors:           restore.Errors,
		Results:          results,
		ValidationErrors: state.Status.ValidationErrors,
		Options: api.RestoreOptions{
			IncludedNamespaces:      state.Spec.IncludedNamespaces,
			IncludedResources:       state.Spec.IncludedResources,
			IncludeClusterResources: state.Spec.IncludeClusterResources,
			ExcludedNamespaces:      state.Spec.ExcludedNamespaces,
			ExcludedResources:       state.Spec.ExcludedResources,
			LabelSelector:           state.Spec.LabelSelector,
			NamespaceMapping:        state.Spec.NamespaceMapping,
			RestorePVs:              state.Spec.RestorePVs,
		},
	}

	if restore.Bucket.ID > 0 {
		item.Bucket = restore.Bucket.ConvertModelToEntity()
	}

	return item
}

// GetState unmarshals a stored state JSON into arkAPI.Restore
func (restore *ClusterBackupRestoresModel) GetState() *arkAPI.Restore {

	var state *arkAPI.Restore
	err := json.Unmarshal(restore.State, &state)
	if err != nil {
		return nil
	}

	return state
}

// GetResults unmarshals a stored result JSON into api.RestoreResults
func (restore *ClusterBackupRestoresModel) GetResults() *api.RestoreResults {

	var results *api.RestoreResults
	err := json.Unmarshal(restore.Results, &results)
	if err != nil {
		return nil
	}

	return results
}
