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
	"fmt"
	"time"

	arkAPI "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/ark/api"
)

// CreateRestore creates an ARK restore by a CreateRestoreRequest
func (c *Client) CreateRestore(req api.CreateRestoreRequest) (*arkAPI.Restore, error) {
	name := fmt.Sprintf("%s-%s", req.BackupName, time.Now().Format("20060102150405"))

	restore := arkAPI.Restore{
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

	err := c.Client.Create(context.Background(), &restore)
	if err != nil {
		return nil, err
	}

	err = c.Client.Get(context.Background(), types.NamespacedName{
		Name:      restore.Name,
		Namespace: restore.Namespace,
	}, &restore)
	if err != nil {
		return nil, err
	}

	return &restore, nil
}

// ListRestores lists ARK restores
func (c *Client) ListRestores() (*arkAPI.RestoreList, error) {
	var restores arkAPI.RestoreList

	err := c.Client.List(context.Background(), &restores, runtimeclient.InNamespace(c.Namespace))
	if err != nil {
		return nil, err
	}

	return &restores, nil
}

// GetRestoreByName gets an ARK restore by name
func (c *Client) GetRestoreByName(name string) (*arkAPI.Restore, error) {
	var restore arkAPI.Restore

	err := c.Client.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace,
	}, &restore)
	if err != nil {
		return nil, err
	}

	return &restore, nil
}

// DeleteRestoreByName deletes an ARK restore by name
func (c *Client) DeleteRestoreByName(name string) error {
	restore, err := c.GetRestoreByName(name)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return c.Client.Delete(context.Background(), restore)
}
