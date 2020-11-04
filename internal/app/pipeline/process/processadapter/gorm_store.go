// Copyright © 2020 Banzai Cloud
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

package processadapter

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/process"
)

// TableName constants
const (
	processTableName      = "processes"
	processEventTableName = "process_events"
)

type processModel struct {
	ID           string              `gorm:"primary_key"`
	ParentID     string              `gorm:"index"`
	OrgID        uint                `gorm:"not null"`
	Type         string              `gorm:"not null"`
	Log          string              `gorm:"type:text"`
	ResourceID   string              `gorm:"not null"`
	ResourceType string              `gorm:"not null"`
	Status       string              `gorm:"not null"`
	StartedAt    time.Time           `gorm:"index:idx_start_time_end_time;default:current_timestamp;not null"`
	FinishedAt   *time.Time          `gorm:"index:idx_start_time_end_time"`
	Events       []processEventModel `gorm:"foreignkey:ProcessID"`
}

// TableName changes the default table name.
func (processModel) TableName() string {
	return processTableName
}

type processEventModel struct {
	ID        uint      `gorm:"auto_increment,primary_key"`
	ProcessID string    `gorm:"not null"`
	Type      string    `gorm:"not null"`
	Log       string    `gorm:"type:text"`
	Status    string    `gorm:"not null"`
	Timestamp time.Time `gorm:"default:current_timestamp;not null"`
}

// TableName changes the default table name.
func (processEventModel) TableName() string {
	return processEventTableName
}

// GormStore is a notification store using Gorm for data persistence.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore returns a new GormStore.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{
		db: db,
	}
}

// GetProcess returns the list of active processes.
func (s *GormStore) GetProcess(ctx context.Context, id string) (process.Process, error) {
	pm := processModel{
		ID: id,
	}

	err := s.db.First(&pm, pm).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return process.Process{}, process.NotFoundError{ID: id}
		}
		return process.Process{}, errors.Wrap(err, "failed to find process")
	}

	var processEvents []processEventModel

	err = s.db.Model(&pm).Related(&processEvents, "Events").Error
	if err != nil {
		return process.Process{}, errors.Wrap(err, "failed to find process events")
	}

	p := process.Process{
		Id:           pm.ID,
		ParentId:     pm.ParentID,
		OrgId:        int32(pm.OrgID),
		Type:         pm.Type,
		Log:          pm.Log,
		StartedAt:    pm.StartedAt,
		FinishedAt:   pm.FinishedAt,
		ResourceId:   pm.ResourceID,
		ResourceType: pm.ResourceType,
		Status:       process.ProcessStatus(pm.Status),
	}

	for _, em := range processEvents {
		p.Events = append(p.Events, process.ProcessEvent{
			Id:        int32(em.ID),
			ProcessId: em.ProcessID,
			Type:      em.Type,
			Log:       em.Log,
			Status:    process.ProcessStatus(em.Status),
			Timestamp: em.Timestamp,
		})
	}

	return p, nil
}

// ListProcesses returns the list of active processes.
func (s *GormStore) ListProcesses(ctx context.Context, query process.Process) ([]process.Process, error) {
	var processes []processModel

	err := s.db.Find(&processes, query).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to find processes")
	}

	result := []process.Process{}

	for _, pm := range processes {
		p := process.Process{
			Id:           pm.ID,
			ParentId:     pm.ParentID,
			OrgId:        int32(pm.OrgID),
			Log:          pm.Log,
			StartedAt:    pm.StartedAt,
			FinishedAt:   pm.FinishedAt,
			ResourceId:   pm.ResourceID,
			ResourceType: pm.ResourceType,
			Type:         pm.Type,
			Status:       process.ProcessStatus(pm.Status),
		}

		var processEvents []processEventModel

		err := s.db.Model(&pm).Related(&processEvents, "Events").Error
		if err != nil {
			return nil, errors.Wrap(err, "failed to find process events")
		}

		for _, em := range processEvents {
			p.Events = append(p.Events, process.ProcessEvent{
				ProcessId: em.ProcessID,
				Type:      em.Type,
				Log:       em.Log,
				Status:    process.ProcessStatus(em.Status),
				Timestamp: em.Timestamp,
			})
		}

		result = append(result, p)
	}

	return result, nil
}

// LogProcess logs a process entry
func (s *GormStore) LogProcess(ctx context.Context, p process.Process) error {
	existing := processModel{ID: p.Id, ParentID: p.ParentId}

	err := s.db.Find(&existing).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			pm := processModel{
				ID:           p.Id,
				ParentID:     p.ParentId,
				OrgID:        uint(p.OrgId),
				Log:          p.Log,
				Type:         p.Type,
				ResourceID:   p.ResourceId,
				ResourceType: p.ResourceType,
				Status:       string(p.Status),
				StartedAt:    p.StartedAt,
			}

			err := s.db.Create(&pm).Error
			return errors.Wrap(err, "failed to create process")
		}

		return err
	}

	existing.Log = p.Log
	existing.Status = string(p.Status)
	existing.FinishedAt = p.FinishedAt

	err = s.db.Save(&existing).Error
	if err != nil {
		return errors.Wrap(err, "failed to update process")
	}

	return nil
}

// LogProcessEvent logs a process event
func (s *GormStore) LogProcessEvent(ctx context.Context, p process.ProcessEvent) error {
	pem := processEventModel{
		ProcessID: p.ProcessId,
		Type:      p.Type,
		Log:       p.Log,
		Status:    string(p.Status),
		Timestamp: p.Timestamp,
	}

	err := s.db.Create(&pem).Error
	return errors.Wrap(err, "failed to create process event")
}
