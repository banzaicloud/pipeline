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

package k8sutil

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnsureNamespace creates a namespace on a cluster if it does not exist.
func EnsureNamespace(client kubernetes.Interface, namespace string) error {
	return EnsureNamespaceWithLabel(client, namespace, nil)
}

// EnsureNamespaceWithLabel creates a namespace with optional labels
func EnsureNamespaceWithLabel(client kubernetes.Interface, namespace string, labels map[string]string) error {
	// add label to mark the namespace that it was created by Pipeline
	mergedLabels := map[string]string{}
	for k, v := range labels {
		mergedLabels[k] = v
	}

	ownerLabel := "owner"
	ownerLabelValue := "pipeline"

	if v, ok := mergedLabels[ownerLabel]; ok && v != ownerLabelValue {
		return errors.WithStack(fmt.Errorf("%q namespace label is reserved for internal use", ownerLabel))
	}
	mergedLabels[ownerLabel] = ownerLabelValue

	_, err := client.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: mergedLabels,
		},
	}, metav1.CreateOptions{})

	if err != nil && k8sapierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return errors.WrapIf(err, "failed to create namespace")
	}

	return nil
}
