// Copyright © 2020 Banzai Cloud
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sHealthChecker struct {
	namespaces []string
}

func MakeK8sHealthChecker(namespaces []string) K8sHealthChecker {
	return K8sHealthChecker{
		namespaces: namespaces,
	}
}

func (c K8sHealthChecker) Check(ctx context.Context, organizationID uint, clusterName string, client kubernetes.Interface) error {
	nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.WrapIf(err, "could not list nodes")
	}

	if len(nodeList.Items) == 0 {
		return errors.New("nodelist is empty")
	}

	for _, node := range nodeList.Items {
		if err := checkNodeStatus(node); err != nil {
			return errors.WrapIf(err, "not all nodes are Ready")
		}
	}

	// TODO namespces to check system pods

	return nil
}

func checkNodeStatus(node corev1.Node) error {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status != corev1.ConditionTrue {
				return errors.NewWithDetails("node is not Ready", map[string]interface{}{
					"node":      node.Name,
					"condition": condition.Status,
				})
			}
		}
	}

	return nil
}

// func checkSystemPods() error {

// 	return nil
// }
