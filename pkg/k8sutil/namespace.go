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
	"encoding/json"
	"time"

	"github.com/banzaicloud/pipeline/internal/backoff"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// EnsureNamespaceWithLabelWithRetry creates a namespace with optional labels and retries if fails because of etcd timeout
func EnsureNamespaceWithLabelWithRetry(client kubernetes.Interface, namespace string, labels map[string]string) (err error) {
	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      10 * time.Second,
		MaxRetries: 5,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)
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

// LabelNamespaceIgnoreNotFound patches a namespace by adding new labels, returns without error if namespace is not found
func LabelNamespaceIgnoreNotFound(log logrus.FieldLogger, client kubernetes.Interface, namespace string, labels map[string]string) error {

	log.Debugf("labels to add: %v", labels)

	ns, err := client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		if k8sapierrors.IsNotFound(err) {
			log.Warnf("namespace not found")
			return nil
		}
		return emperror.Wrap(err, "failed to get namespace to label")
	}

	var patch []patchOperation

	if len(ns.Labels) == 0 {
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  "/metadata/labels",
			Value: labels,
		})
	} else {
		// need to create a patch for every label, otherwise it would delete all existing labels and only add the new ones
		for l, v := range labels {
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  "/metadata/labels/" + l,
				Value: v,
			})
		}
	}

	log.Debugf("patch: %v", patch)

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return emperror.Wrap(err, "failed to label namespace")
	}

	_, err = client.CoreV1().Namespaces().Patch(namespace, types.JSONPatchType, patchBytes)
	if k8sapierrors.IsNotFound(err) {
		log.Warnf("namespace not found")
	} else if err != nil {
		log.Warnf(err.Error())
		return emperror.Wrap(err, "failed to label namespace")
	}
	return nil
}
