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
	"context"

	arkAPI "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/ark/api"
)

func (c *Client) getScheduleSpec(req *api.CreateScheduleRequest) arkAPI.ScheduleSpec {
	return arkAPI.ScheduleSpec{
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
	}
}

// CreateOrUpdateSchedule creates an ARK schedule by a CreateScheduleRequest
func (c *Client) CreateOrUpdateSchedule(req *api.CreateScheduleRequest) error {

	schedule := arkAPI.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.Namespace,
			Name:      req.Name,
			Labels:    req.Labels,
		},
	}

	err := c.Client.Get(context.Background(), types.NamespacedName{
		Name:      req.Name,
		Namespace: c.Namespace,
	}, &schedule)

	notFound := false
	if err != nil {
		if k8serrors.IsNotFound(err) {
			notFound = true
		} else {
			return err
		}
	}

	if notFound {
		schedule.Spec = c.getScheduleSpec(req)
		err = c.Client.Create(context.Background(), &schedule)
		if err != nil {
			return err
		}
	} else {
		schedule.Labels = req.Labels
		schedule.Spec = c.getScheduleSpec(req)
		err = c.Client.Update(context.Background(), &schedule)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListSchedules lists ARK schedules
func (c *Client) ListSchedules() (*arkAPI.ScheduleList, error) {
	var schedules arkAPI.ScheduleList

	err := c.Client.List(context.Background(), &schedules, runtimeclient.InNamespace(c.Namespace))
	if err != nil {
		return nil, err
	}

	return &schedules, nil
}

// GetScheduleByName gets an ARK schedule by name
func (c *Client) GetScheduleByName(name string) (*arkAPI.Schedule, error) {
	var schedule arkAPI.Schedule

	err := c.Client.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace,
	}, &schedule)
	if err != nil {
		return nil, err
	}

	return &schedule, nil
}

// DeleteScheduleByName deletes a schedule by name
func (c *Client) DeleteScheduleByName(name string) error {
	schedule, err := c.GetScheduleByName(name)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return c.Client.Delete(context.Background(), schedule)
}
