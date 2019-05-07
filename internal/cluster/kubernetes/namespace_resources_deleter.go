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
	"github.com/sirupsen/logrus"
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
		DeleteCollectioner interface {
			DeleteCollection(*metav1.DeleteOptions, metav1.ListOptions) error
		}
		Name string
	}{
		{client.AppsV1().Deployments(namespace), "Deployments"},
		{client.AppsV1().DaemonSets(namespace), "DaemonSets"},
		{client.AppsV1().StatefulSets(namespace), "StatefulSets"},
		{client.AppsV1().ReplicaSets(namespace), "ReplicaSets"},
		{client.CoreV1().Pods(namespace), "Pods"},
		{client.CoreV1().PersistentVolumeClaims(namespace), "PersistentVolumeClaims"},
	}

	for _, resourceType := range resourceTypes {
		err := retry(func() error {
			logger.Debugf("deleting %s", resourceType.Name)
			err := resourceType.DeleteCollectioner.DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			if err != nil {
				logger.Infof("could not delete %s: %v", resourceType.Name, err)
			}
			return err
		}, 6, 1)
		if err != nil {
			return emperror.Wrapf(err, "could not delete %s", resourceType.Name)
		}
	}

	return nil
}
