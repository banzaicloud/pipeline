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
	"time"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/src/cluster"
)

// waitForNamespaceBeDeleted wait for a k8s namespace to be deleted
func (m *MeshReconciler) waitForNamespaceBeDeleted(client runtimeclient.Client, name string) error {
	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	var namespace corev1.Namespace
	err := backoff.Retry(func() error {
		err := client.Get(context.Background(), types.NamespacedName{
			Name: name,
		}, &namespace)
		if k8serrors.IsNotFound(err) {
			return nil
		}

		return errors.New("namespace still exists")
	}, backoffPolicy)

	return errors.WithStack(err)
}

func (m *MeshReconciler) reconcileNamespace(namespace string, desiredState DesiredState, c cluster.CommonCluster, labels map[string]string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: labels,
		},
	}

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, ns))
	}

	err = m.deleteResource(client, ns)
	if err != nil {
		return errors.WithStack(err)
	}

	m.logger.WithField("namespace", namespace).Debug("waiting for namespace to be deleted")

	return errors.WrapIfWithDetails(m.waitForNamespaceBeDeleted(client, namespace), "timeout during waiting for namespace to be deleted", "namespace", namespace)
}
