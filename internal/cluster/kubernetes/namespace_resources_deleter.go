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
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceResourcesDeleter struct {
	logger logrus.FieldLogger
}

func MakeNamespaceResourcesDeleter(logger logrus.FieldLogger) NamespaceResourcesDeleter {
	return NamespaceResourcesDeleter{
		logger: logger,
	}
}

func (d NamespaceResourcesDeleter) Delete(organizationID uint, clusterName string, k8sConfig []byte, namespace string) error {
	logger := d.logger.WithField("organizationID", organizationID).WithField("clusterName", clusterName).WithField("namespace", namespace)

	client, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return err
	}
	resourceTypes := []struct {
		Resource interface {
			DeleteCollection(*metav1.DeleteOptions, metav1.ListOptions) error
		}
		Name  string
		Check func(interface{}) error
	}{
		{client.AppsV1().Deployments(namespace), "Deployments", nil},
		{client.AppsV1().DaemonSets(namespace), "DaemonSets", nil},
		{client.AppsV1().StatefulSets(namespace), "StatefulSets", nil},
		{client.AppsV1().ReplicaSets(namespace), "ReplicaSets", nil},
		{client.CoreV1().Pods(namespace), "Pods", nil},
		{client.CoreV1().PersistentVolumeClaims(namespace),
			"PersistentVolumeClaims",
			func(pvcs interface{}) error {
				res, err := pvcs.(interface {
					List(opts metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error)
				}).List(metav1.ListOptions{})
				if err != nil {
					return err
				}
				if len(res.Items) > 0 {
					return fmt.Errorf("PVC remained present after deletion: %v", res.Items)
				}
				return nil
			}},
	}

	for _, resourceType := range resourceTypes {
		err := retry(func() error {
			logger.Debugf("deleting %s", resourceType.Name)
			err := resourceType.Resource.DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			if err != nil {
				logger.Infof("could not delete %s: %v", resourceType.Name, err)
			}
			return err
		}, 6, 1)
		if err != nil {
			return emperror.Wrapf(err, "could not delete %s", resourceType.Name)
		}
		if resourceType.Check != nil {
			err := retry(func() error { return resourceType.Check(resourceType.Resource) }, 10, 20)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
