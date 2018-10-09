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
	"fmt"
	"time"

	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// CreateRestore creates an ARK restore by a CreateRestoreRequest
func (c *Client) CreateRestore(req api.CreateRestoreRequest) (*arkAPI.Restore, error) {

	name := fmt.Sprintf("%s-%s", req.BackupName, time.Now().Format("20060102150405"))

	restore := &arkAPI.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.Namespace,
			Name:      name,
			Labels:    req.Labels,
		},
		Spec: arkAPI.RestoreSpec{
			BackupName:              req.BackupName,
			IncludedNamespaces:      req.Options.IncludedNamespaces,
			IncludedResources:       req.Options.IncludedResources,
			ExcludedNamespaces:      req.Options.ExcludedNamespaces,
			ExcludedResources:       req.Options.ExcludedResources,
			IncludeClusterResources: req.Options.IncludeClusterResources,
			LabelSelector:           req.Options.LabelSelector,
			RestorePVs:              req.Options.RestorePVs,
		},
	}

	restore, err := c.Client.ArkV1().Restores(restore.Namespace).Create(restore)
	if err != nil {
		return nil, err
	}

	return restore, nil
}

// ListRestores lists ARK restores
func (c *Client) ListRestores(listOptions metav1.ListOptions) (restores *arkAPI.RestoreList, err error) {

	restores, err = c.Client.ArkV1().Restores(c.Namespace).List(listOptions)

	return
}

// GetRestoreByName gets an ARK restore by name
func (c *Client) GetRestoreByName(name string) (restore *arkAPI.Restore, err error) {

	restore, err = c.Client.Ark().Restores(c.Namespace).Get(name, metav1.GetOptions{})

	return
}

// DeleteRestoreByName deletes an ARK restore by name
func (c *Client) DeleteRestoreByName(name string) error {

	return c.Client.Ark().Restores(c.Namespace).Delete(name, &metav1.DeleteOptions{})
}
