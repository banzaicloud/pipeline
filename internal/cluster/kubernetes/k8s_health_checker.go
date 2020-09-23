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

type HealthChecker struct {
	namespaces []string
}

func MakeHealthChecker(namespaces []string) HealthChecker {
	return HealthChecker{
		namespaces: namespaces,
	}
}

func (c HealthChecker) Check(ctx context.Context, client kubernetes.Interface) error {
	// TODO pagination
	nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.WrapIf(err, "could not list nodes")
	}

	if len(nodeList.Items) < 1 {
		return errors.New("nodelist is empty")
	}

	for _, node := range nodeList.Items {
		if err := checkNodeStatus(node); err != nil {
			return err
		}
	}

	// TODO namespces to check system pods
	for _, namespace := range c.namespaces {
		// TODO pagination
		podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not list pods", map[string]interface{}{
				"namespace": namespace,
			})
		}
		if err := checkSystemPods(podList); err != nil {
			return errors.WrapIfWithDetails(err, "not all pods are ready", map[string]interface{}{
				"namespace": namespace,
			})
		}
	}

	return nil
}

func checkNodeStatus(node corev1.Node) error {
	for _, condition := range node.Status.Conditions {
		if condition.Type != corev1.NodeReady {
			continue
		}
		if condition.Status != corev1.ConditionTrue {
			return errors.NewWithDetails("node is not Ready", map[string]interface{}{
				"node":      node.Name,
				"condition": condition.Status,
			})
		}
	}

	return nil
}

func checkSystemPods(podList *corev1.PodList) error {
	if len(podList.Items) < 1 {
		return errors.New("podlist is empty")
	}

	// TODO check system pods are exist, check status of daemonsets?

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return errors.NewWithDetails("pod is not Running", map[string]interface{}{
				"pod":   pod.Name,
				"phase": pod.Status.Phase,
			})
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type != corev1.PodReady {
				continue
			}
			if condition.Status != corev1.ConditionTrue {
				return errors.NewWithDetails("pod is not Ready", map[string]interface{}{
					"pod":       pod.Name,
					"condition": condition.Status,
				})
			}
		}
	}

	return nil
}
