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
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UserNamespaceDeleter struct {
	logger logrus.FieldLogger
}

func MakeUserNamespaceDeleter(logger logrus.FieldLogger) UserNamespaceDeleter {
	return UserNamespaceDeleter{
		logger: logger,
	}
}

func (d UserNamespaceDeleter) Delete(k8sConfig []byte) error {
	client, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create k8s client")
	}
	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return emperror.Wrap(err, "could not list namespaces to delete")
	}

	for _, ns := range namespaces.Items {
		switch ns.Name {
		case "default", "kube-system", "kube-public":
			continue
		}
		err := retry(func() error {
			d.logger.Infof("deleting kubernetes namespace %q", ns.Name)
			err := client.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{})
			if err != nil {
				return emperror.Wrapf(err, "failed to delete %q namespace", ns.Name)
			}
			return nil
		}, 3, 1)
		if err != nil {
			return err
		}
	}
	err = retry(func() error {
		namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not list remaining namespaces")
		}
		left := []string{}
		for _, ns := range namespaces.Items {
			if len(ns.Spec.Finalizers) > 0 {
				d.logger.Infof("can't delete namespace %q with finalizers %s", ns.Name, ns.Spec.Finalizers)
				continue
			}

			switch ns.Name {
			case "default", "kube-system", "kube-public":
				continue
			default:
				d.logger.Infof("namespace %q still %s", ns.Name, ns.Status)
				left = append(left, ns.Name)
			}
		}
		if len(left) > 0 {
			return emperror.With(errors.Errorf("namespaces remaining after deletion: %v", left), "namespaces", left)
		}
		return nil
	}, 20, 30)
	return err
}
