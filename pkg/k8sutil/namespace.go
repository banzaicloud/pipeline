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
	"fmt"
	"time"

	"emperror.dev/emperror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/pkg/backoff"
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

	_, err := client.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: mergedLabels,
		},
	})

	if err != nil && k8sapierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return emperror.Wrap(err, "failed to create namespace")
	}

	return nil
}

// EnsureNamespaceWithLabelWithRetry creates a namespace with optional labels and retries if fails because of etcd timeout
func EnsureNamespaceWithLabelWithRetry(client kubernetes.Interface, namespace string, labels map[string]string) (err error) {
	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      10 * time.Second,
		MaxRetries: 5,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)
	err = backoff.Retry(func() error {
		if err := EnsureNamespaceWithLabel(client, namespace, labels); err != nil {
			if IsK8sErrorPermanent(err) {
				return backoff.MarkErrorPermanent(err)
			}
		}
		return nil
	}, backoffPolicy)
	return
}

// EnsureLabelsOnNamespace adds a set of labels a namespace, overwriting existing label keys
func EnsureLabelsOnNamespace(client kubernetes.Interface, namespace string, labels map[string]string) error {

	namespaceObj, err := client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		return emperror.Wrap(err, "failed to get namespace")
	}

	if namespaceObj.Labels == nil {
		namespaceObj.Labels = make(map[string]string, len(labels))
	}

	for k, v := range labels {
		namespaceObj.Labels[k] = v
	}

	_, err = client.CoreV1().Namespaces().Update(namespaceObj)
	if err != nil && k8sapierrors.IsAlreadyExists(err) {
		return emperror.Wrap(err, "failed to update namespace")
	}

	return nil
}
