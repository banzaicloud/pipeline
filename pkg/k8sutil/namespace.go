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
	"github.com/goph/emperror"
	"k8s.io/api/core/v1"
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
	_, err := client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: labels,
		},
	})

	if err != nil && k8sapierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return emperror.Wrap(err, "failed to create namespace")
	}

	return nil
}
