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
	"github.com/sirupsen/logrus"
	arkAPI "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/ark/client"
)

// SchedulesService is for managing ARK schedules
type SchedulesService struct {
	arkClientService client.ClientService
	logger           logrus.FieldLogger
}

// SchedulesServiceFactory creates and returns an initialized SchedulesService instance
func SchedulesServiceFactory(
	arkClientService client.ClientService,
	logger logrus.FieldLogger,
) *SchedulesService {
	return NewSchedulesService(arkClientService, logger)
}

// NewSchedulesService creates and returns an initialized SchedulesService instance
func NewSchedulesService(
	arkClientService client.ClientService,
	logger logrus.FieldLogger,
) *SchedulesService {
	return &SchedulesService{
		arkClientService: arkClientService,
		logger:           logger,
	}
}

// Create creates a schedule by a CreateBackupRequest
func (s *SchedulesService) CreateOrUpdateSchedule(backupRequest *api.CreateBackupRequest, schedule string) error {
	req := &api.CreateScheduleRequest{
		Name:     backupRequest.Name,
		TTL:      backupRequest.TTL,
		Labels:   backupRequest.Labels,
		Schedule: schedule,
		Options:  backupRequest.Options,
	}

	client, err := s.arkClientService.GetClient()
	if err != nil {
		return err
	}

	return client.CreateOrUpdateSchedule(req)
}

// GetByName gets a schedule by name
func (s *SchedulesService) GetByName(name string) (*api.Schedule, error) {
	client, err := s.arkClientService.GetClient()
	if err != nil {
		return nil, err
	}

	schedule, err := client.GetScheduleByName(name)
	if err != nil {
		return nil, err
	}

	return s.convertScheduleToEntity(*schedule), nil
}

// DeleteByName deletes a schedule by name
func (s *SchedulesService) DeleteByName(name string) error {
	client, err := s.arkClientService.GetClient()
	if err != nil {
		return err
	}

	err = client.DeleteScheduleByName(name)
	if err != nil {
		return err
	}

	return nil
}

// List gets all schedule
func (s *SchedulesService) List() (schedules []*api.Schedule, err error) {
	schedules = make([]*api.Schedule, 0)

	client, err := s.arkClientService.GetClient()
	if err != nil {
		return nil, err
	}

	list, err := client.ListSchedules()
	if err != nil {
		return nil, err
	}

	for _, item := range list.Items {
		schedule := s.convertScheduleToEntity(item)
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

func (s *SchedulesService) convertScheduleToEntity(schedule arkAPI.Schedule) *api.Schedule {
	sched := &api.Schedule{
		UID:              string(schedule.GetUID()),
		Name:             schedule.Name,
		Schedule:         schedule.Spec.Schedule,
		TTL:              schedule.Spec.Template.TTL,
		Labels:           schedule.ObjectMeta.Labels,
		Status:           string(schedule.Status.Phase),
		ValidationErrors: schedule.Status.ValidationErrors,
		Options: api.BackupOptions{
			IncludedNamespaces:      schedule.Spec.Template.IncludedNamespaces,
			IncludedResources:       schedule.Spec.Template.IncludedResources,
			IncludeClusterResources: schedule.Spec.Template.IncludeClusterResources,
			ExcludedNamespaces:      schedule.Spec.Template.ExcludedNamespaces,
			ExcludedResources:       schedule.Spec.Template.ExcludedResources,
			LabelSelector:           schedule.Spec.Template.LabelSelector,
			SnapshotVolumes:         schedule.Spec.Template.SnapshotVolumes,
		},
	}
	if schedule.Status.LastBackup != nil {
		sched.LastBackup = schedule.Status.LastBackup.Time
	}
	return sched
}
