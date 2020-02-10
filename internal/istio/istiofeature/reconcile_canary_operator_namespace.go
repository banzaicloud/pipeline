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
	"context"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

func (m *MeshReconciler) ReconcileCanaryOperatorNamespace(desiredState DesiredState) error {
	m.logger.Debug("reconciling canary operator namespace")
	defer m.logger.Debug("canary operator namespace reconciled")

	client, err := m.getMasterK8sClient()
	if err != nil {
		return err
	}

	k8sclient, err := m.getK8sClient(m.Master)
	if err != nil {
		return err
	}

	var namespace corev1.Namespace
	if desiredState == DesiredStatePresent {
		err := client.Get(context.Background(), types.NamespacedName{
			Name: canaryOperatorNamespace,
		}, &namespace)
		if k8serrors.IsNotFound(err) {
			err := k8sutil.EnsureNamespaceWithLabelWithRetry(k8sclient, canaryOperatorNamespace,
				map[string]string{
					"istio-injection": "enabled",
				})
			if err != nil {
				return errors.WrapIf(err, "could not create canary operator namespace")
			}
		} else if err != nil {
			return errors.WrapIf(err, "could not get canary operator namespace")
		}
	} else {
		err := client.Get(context.Background(), types.NamespacedName{
			Name: canaryOperatorNamespace,
		}, &namespace)
		if k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			return errors.WrapIf(err, "could not get canary operator namespace")
		}

		err = client.Delete(context.Background(), &namespace)
		if err != nil {
			return errors.WrapIf(err, "could not delete canary operator namespace")
		}

		m.logger.Debug("waiting for canary operator namespace to be deleted")
		err = m.waitForNamespaceBeDeleted(client, canaryOperatorNamespace)
		if err != nil {
			return errors.WrapIf(err, "timeout during waiting for canary operator namespace to be deleted")
		}
	}

	return nil
}
