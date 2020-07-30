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

package kubernetes

import (
	"context"

	"emperror.dev/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

type UserNamespaceDeleter struct {
	logger logrus.FieldLogger
}

func MakeUserNamespaceDeleter(logger logrus.FieldLogger) UserNamespaceDeleter {
	return UserNamespaceDeleter{
		logger: logger,
	}
}

func (d UserNamespaceDeleter) Delete(ctx context.Context, organizationID uint, clusterName string, namespaces *corev1.NamespaceList, k8sConfig []byte) ([]string, error) {
	logger := d.logger.WithField("organizationID", organizationID).WithField("clusterName", clusterName)

	client, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create k8s client")
	}

	err = retry(func() error {
		nsList := namespaces
		if nsList == nil {
			nsList, err = client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return errors.WrapIf(err, "could not list namespaces")
			}
		}

		for _, ns := range nsList.Items {
			logger := logger.WithField("namespace", ns.Name)
			switch ns.Name {
			case "default", "kube-system", "kube-public":
				continue
			}
			logger.Info("deleting kubernetes namespace")
			err := client.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{})
			if err != nil && !k8sapierrors.IsNotFound(err) {
				// check if the namespace transitioned into 'Terminating' phase, if so
				// ignore the error
				namespace, errGet := client.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{})
				if errGet != nil {
					return errors.WrapIff(err, "could not get %q namespace details (%v) after failed deletion", ns.Name, errGet)
				}

				if namespace.Status.Phase != corev1.NamespaceTerminating {
					return errors.WrapIff(err, "failed to delete %q namespace", ns.Name)
				}
			}
		}
		return nil
	}, 3, 1)
	if err != nil {
		return nil, err
	}

	var left, gaveUp []string
	err = retry(func() error {
		remainingNamespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return errors.WrapIf(err, "could not list remaining namespaces")
		}

		left = nil
		gaveUp = nil
		for _, remainingNamespace := range remainingNamespaces.Items {
			switch remainingNamespace.Name {
			case "default", "kube-system", "kube-public":
				continue
			}
			if namespaces != nil && len(namespaces.Items) > 0 {
				match := false
				for _, ns := range namespaces.Items {
					if remainingNamespace.Name == ns.Name {
						match = true
						break
					}
				}

				if !match {
					continue
				}
			}

			if len(remainingNamespace.Spec.Finalizers) > 0 {
				d.logger.Infof("can't delete namespace %q with finalizers %s", remainingNamespace.Name, remainingNamespace.Spec.Finalizers)
				gaveUp = append(gaveUp, remainingNamespace.Name)
				continue
			}

			logger.Infof("namespace %q still %s", remainingNamespace.Name, remainingNamespace.Status)
			left = append(left, remainingNamespace.Name)
		}
		if len(left) > 0 {
			return errors.WithDetails(errors.Errorf("namespaces remaining after deletion: %v", left), "namespaces", left)
		}
		return nil
	}, 20, 30)

	return append(gaveUp, left...), err
}
