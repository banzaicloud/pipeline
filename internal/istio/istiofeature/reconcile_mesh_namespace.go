// Copyright © 2019 Banzai Cloud
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

package istiofeature

import (
	"time"

	"emperror.dev/emperror"
	"github.com/banzaicloud/pipeline/internal/backoff"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (m *MeshReconciler) ReconcileMeshNamespace(desiredState DesiredState) error {
	m.logger.Debug("reconciling mesh namespace")
	defer m.logger.Debug("mesh namespace reconciled")

	client, err := m.getMasterK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		_, err := client.CoreV1().Namespaces().Get(meshNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			err := k8sutil.EnsureNamespaceWithLabelWithRetry(client, meshNamespace,
				map[string]string{
					"istio-injection": "enabled",
				})
			if err != nil {
				return emperror.Wrap(err, "could not create namespace")
			}
		} else if err != nil {
			return emperror.Wrap(err, "could not get namespace")
		}
	} else {
		_, err := client.CoreV1().Namespaces().Get(meshNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			return emperror.Wrap(err, "could not get namespace")
		}

		err = client.CoreV1().Namespaces().Delete(meshNamespace, &metav1.DeleteOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not delete namespace")
		}

		err = m.waitForMeshNamespaceBeDeleted(client)
		if err != nil {
			return emperror.Wrap(err, "timeout during waiting for mesh namespace to be deleted")
		}
	}

	return nil
}

// waitForMeshNamespaceBeDeleted wait for mesh namespace to be deleted
func (m *MeshReconciler) waitForMeshNamespaceBeDeleted(client *kubernetes.Clientset) error {
	m.logger.Debug("waiting for mesh namespace to be deleted")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	err := backoff.Retry(func() error {
		_, err := client.CoreV1().Namespaces().Get(meshNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}

		return errors.New("mesh namespace still exists")
	}, backoffPolicy)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
