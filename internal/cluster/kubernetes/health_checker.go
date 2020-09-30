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
	"time"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/pkg/backoff"
)

const (
	backoffDelay      = 5 * time.Second
	backoffMaxRetries = 10
)

// HealthChecker implements Check
type HealthChecker struct {
	namespaces []string
}

// NewHealthChecker returns HealthChecker
func NewHealthChecker(namespaces []string) HealthChecker {
	return HealthChecker{
		namespaces: namespaces,
	}
}

// Check cheks nodes and system pods
func (c HealthChecker) Check(ctx context.Context, client kubernetes.Interface) error {
	nodeList, err := listNodes(ctx, client)
	if err != nil {
		return err
	}

	if len(nodeList.Items) < 1 {
		return errors.New("nodelist is empty")
	}

	for _, node := range nodeList.Items {
		if err := checkNodeStatus(node); err != nil {
			return err
		}
	}

	for _, namespace := range c.namespaces {
		podList, err := listSystemPods(ctx, client, namespace)
		if err != nil {
			return err
		}

		if err := checkPodStatus(podList); err != nil {
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

func checkPodStatus(podList *corev1.PodList) error {
	if len(podList.Items) < 1 {
		return errors.New("podlist is empty")
	}

	for _, pod := range podList.Items {
		if !(pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded) {
			return errors.NewWithDetails("pod is not Running or Succeeded", map[string]interface{}{
				"pod":   pod.Name,
				"phase": pod.Status.Phase,
			})
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type != corev1.PodReady {
				continue
			}
			if condition.Status != corev1.ConditionTrue && pod.Status.Phase != corev1.PodSucceeded {
				return errors.NewWithDetails("pod is not Ready", map[string]interface{}{
					"pod":       pod.Name,
					"condition": condition.Status,
				})
			}
		}
	}

	return nil
}

func listNodes(ctx context.Context, client kubernetes.Interface) (*corev1.NodeList, error) {
	backoffConfig := backoff.ConstantBackoffConfig{
		Delay:      backoffDelay,
		MaxRetries: backoffMaxRetries,
	}
	backoffPolicy := backoff.NewConstantBackoffPolicy(backoffConfig)
	var nodeList *corev1.NodeList

	return nodeList, backoff.Retry(func() error {
		var err error
		nodeList, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return errors.WrapIf(err, "could not list nodes")
		}

		return nil
	}, backoffPolicy)
}

func listSystemPods(ctx context.Context, client kubernetes.Interface, namespace string) (*corev1.PodList, error) {
	backoffConfig := backoff.ConstantBackoffConfig{
		Delay:      backoffDelay,
		MaxRetries: backoffMaxRetries,
	}
	backoffPolicy := backoff.NewConstantBackoffPolicy(backoffConfig)
	var podList *corev1.PodList

	return podList, backoff.Retry(func() error {
		var err error
		podList, err = client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not list pods", map[string]interface{}{
				"namespace": namespace,
			})
		}

		return nil
	}, backoffPolicy)
}
