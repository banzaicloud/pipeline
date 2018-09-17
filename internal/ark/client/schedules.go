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

package client

import (
	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// CreateSchedule creates an ARK schedule by a CreateScheduleRequest
func (c *Client) CreateSchedule(req *api.CreateScheduleRequest) error {

	s := &arkAPI.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.Namespace,
			Name:      req.Name,
			Labels:    req.Labels,
		},
		Spec: arkAPI.ScheduleSpec{
			Template: arkAPI.BackupSpec{
				IncludedNamespaces:      req.Options.IncludedNamespaces,
				ExcludedNamespaces:      req.Options.ExcludedNamespaces,
				IncludedResources:       req.Options.IncludedResources,
				ExcludedResources:       req.Options.ExcludedResources,
				LabelSelector:           req.Options.LabelSelector,
				IncludeClusterResources: req.Options.IncludeClusterResources,
				SnapshotVolumes:         req.Options.SnapshotVolumes,
				TTL:                     req.TTL,
			},
			Schedule: req.Schedule,
		},
	}

	_, err := c.Client.ArkV1().Schedules(s.Namespace).Create(s)
	if err != nil {
		return err
	}

	return nil
}

// ListSchedules lists ARK schedules
func (c *Client) ListSchedules(listOptions metav1.ListOptions) (schedules *arkAPI.ScheduleList, err error) {

	schedules, err = c.Client.ArkV1().Schedules(c.Namespace).List(listOptions)

	return
}

// GetScheduleByName gets an ARK schedule by name
func (c *Client) GetScheduleByName(name string) (schedule *arkAPI.Schedule, err error) {

	schedule, err = c.Client.Ark().Schedules(c.Namespace).Get(name, metav1.GetOptions{})

	return
}

// DeleteScheduleByName deletes a schedule by name
func (c *Client) DeleteScheduleByName(name string) error {

	return c.Client.Ark().Schedules(c.Namespace).Delete(name, &metav1.DeleteOptions{})
}
