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

	"github.com/banzaicloud/pipeline/internal/backoff"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (m *MeshReconciler) ReconcileUistioNamespace(desiredState DesiredState) error {
	m.logger.Debug("reconciling Uistio namespace")
	defer m.logger.Debug("Uistio namespace reconciled")

	client, err := m.GetMasterK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		_, err := client.CoreV1().Namespaces().Get(uistioNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			_, err := client.CoreV1().Namespaces().Create(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: uistioNamespace,
					Labels: map[string]string{
						"istio-injection": "enabled",
					},
				},
			})
			if err != nil {
				return emperror.Wrap(err, "could not create namespace")
			}
		} else if err != nil {
			return emperror.Wrap(err, "could not get namespace")
		}
	} else {
		_, err := client.CoreV1().Namespaces().Get(uistioNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			return emperror.Wrap(err, "could not get namespace")
		}

		err = client.CoreV1().Namespaces().Delete(uistioNamespace, &metav1.DeleteOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not delete namespace")
		}

		err = m.waitForUistioNamespaceBeDeleted(client)
		if err != nil {
			return emperror.Wrap(err, "timeout during waiting for Uistio namespace to be deleted")
		}
	}

	return nil
}

// waitForUistioNamespaceBeDeleted wait for Uistio namespace to be deleted
func (m *MeshReconciler) waitForUistioNamespaceBeDeleted(client *kubernetes.Clientset) error {
	m.logger.Debug("waiting for Uistio namespace to be deleted")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	err := backoff.Retry(func() error {
		_, err := client.CoreV1().Namespaces().Get(uistioNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}

		return errors.New("Uistio namespace still exists")
	}, backoffPolicy)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
