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

	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// CreateBackup creates an ARK backup by a CreateBackupRequest
func (c *Client) CreateBackup(spec api.CreateBackupRequest) (*arkAPI.Backup, error) {
	backup := arkAPI.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.Namespace,
			Name:      spec.Name,
			Labels:    spec.Labels,
		},
		Spec: arkAPI.BackupSpec{
			TTL:                     spec.TTL,
			IncludedNamespaces:      spec.Options.IncludedNamespaces,
			IncludedResources:       spec.Options.IncludedResources,
			ExcludedNamespaces:      spec.Options.ExcludedNamespaces,
			ExcludedResources:       spec.Options.ExcludedResources,
			IncludeClusterResources: spec.Options.IncludeClusterResources,
			LabelSelector:           spec.Options.LabelSelector,
			SnapshotVolumes:         spec.Options.SnapshotVolumes,
		},
	}

	err := c.Client.Create(context.Background(), &backup)
	if err != nil {
		return nil, err
	}

	err = c.Client.Get(context.Background(), types.NamespacedName{
		Name:      backup.Name,
		Namespace: backup.Namespace,
	}, &backup)
	if err != nil {
		return nil, err
	}

	return &backup, nil
}

// ListBackups lists ARK backups
func (c *Client) ListBackups() (*arkAPI.BackupList, error) {
	var backups arkAPI.BackupList

	err := c.Client.List(context.Background(), &backups, runtimeclient.InNamespace(c.Namespace))
	if err != nil {
		return nil, err
	}

	return &backups, nil
}

// GetBackupByName gets an ARK backup by name
func (c *Client) GetBackupByName(name string) (*arkAPI.Backup, error) {
	var backup arkAPI.Backup

	err := c.Client.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace,
	}, &backup)
	if err != nil {
		return nil, err
	}

	return &backup, nil
}

// CreateDeleteBackupRequestByName creates a DeleteBackupRequest for an ARK backup by name
func (c *Client) CreateDeleteBackupRequestByName(name string) error {
	backup, err := c.GetBackupByName(name)
	if err != nil {
		return err
	}

	deleteRequest := arkAPI.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    c.Namespace,
			GenerateName: name + "-",
			Labels: map[string]string{
				arkAPI.BackupNameLabel: backup.Name,
				arkAPI.BackupUIDLabel:  string(backup.UID),
			},
		},
		Spec: arkAPI.DeleteBackupRequestSpec{
			BackupName: backup.Name,
		},
	}

	return c.Client.Create(context.Background(), &deleteRequest)
}
