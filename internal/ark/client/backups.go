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
	v1 "github.com/heptio/ark/pkg/apis/ark/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// CreateBackup creates an ARK backup by a CreateBackupRequest
func (c *Client) CreateBackup(spec api.CreateBackupRequest) (*v1.Backup, error) {

	backup := &arkAPI.Backup{
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

	backup, err := c.Client.ArkV1().Backups(backup.Namespace).Create(backup)
	if err != nil {
		return nil, err
	}

	return backup, nil
}

// ListBackups lists ARK backups
func (c *Client) ListBackups(listOptions metav1.ListOptions) (backups *arkAPI.BackupList, err error) {

	backups, err = c.Client.ArkV1().Backups(c.Namespace).List(listOptions)

	return
}

// GetBackupByName gets an ARK backup by name
func (c *Client) GetBackupByName(name string) (backup *arkAPI.Backup, err error) {

	backup, err = c.Client.Ark().Backups(c.Namespace).Get(name, metav1.GetOptions{})

	return
}

// CreateDeleteBackupRequestByName creates a DeleteBackupRequest for an ARK backup by name
func (c *Client) CreateDeleteBackupRequestByName(name string) (err error) {

	backup, err := c.GetBackupByName(name)
	if err != nil {
		return err
	}

	deleteRequest := &v1.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name + "-",
			Labels: map[string]string{
				v1.BackupNameLabel: backup.Name,
				v1.BackupUIDLabel:  string(backup.UID),
			},
		},
		Spec: v1.DeleteBackupRequestSpec{
			BackupName: backup.Name,
		},
	}

	_, err = c.Client.ArkV1().DeleteBackupRequests(c.Namespace).Create(deleteRequest)

	return
}
