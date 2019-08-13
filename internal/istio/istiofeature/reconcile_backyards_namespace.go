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
	"emperror.dev/emperror"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *MeshReconciler) ReconcileBackyardsNamespace(desiredState DesiredState) error {
	m.logger.Debug("reconciling backyards namespace")
	defer m.logger.Debug("backyards namespace reconciled")

	client, err := m.getMasterK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		_, err := client.CoreV1().Namespaces().Get(backyardsNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			err := k8sutil.EnsureNamespaceWithLabelWithRetry(client, backyardsNamespace,
				map[string]string{
					"istio-injection": "enabled",
				})
			if err != nil {
				return emperror.Wrap(err, "could not create backyards namespace")
			}
		} else if err != nil {
			return emperror.Wrap(err, "could not get backyards namespace")
		}
	} else {
		_, err := client.CoreV1().Namespaces().Get(backyardsNamespace, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			return emperror.Wrap(err, "could not get backyards namespace")
		}

		err = client.CoreV1().Namespaces().Delete(backyardsNamespace, &metav1.DeleteOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not delete backyards namespace")
		}

		m.logger.Debug("waiting for backyards namespace to be deleted")
		err = m.waitForNamespaceBeDeleted(client, backyardsNamespace)
		if err != nil {
			return emperror.Wrap(err, "timeout during waiting for backyards namespace to be deleted")
		}
	}

	return nil
}
