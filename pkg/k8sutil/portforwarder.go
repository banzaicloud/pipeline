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

package k8sutil

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"k8s.io/helm/pkg/kube"
)

type TunnelDestinationNotFoundError error

// New creates a new and initialized tunnel.
func NewKubeTunnel(namespace string, client kubernetes.Interface, config *rest.Config, selector labels.Selector, internalPort int) (*kube.Tunnel, error) {
	podName, err := GetPodName(client.CoreV1(), namespace, selector)
	if err != nil {
		return nil, err
	}
	t := kube.NewTunnel(client.CoreV1().RESTClient(), config, namespace, podName, internalPort)
	return t, t.ForwardPort()
}

// GetTillerPodName fetches the name of tiller pod running in the given namespace.
func GetPodName(client corev1.PodsGetter, namespace string, selector labels.Selector) (string, error) {
	pod, err := getFirstRunningPod(client, namespace, selector)
	if err != nil {
		return "", err
	}
	return pod.ObjectMeta.GetName(), nil
}

func getFirstRunningPod(client corev1.PodsGetter, namespace string, selector labels.Selector) (*v1.Pod, error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := client.Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("could not find pod with labels: %s", selector.String())
	}
	for _, p := range pods.Items {
		if isPodReady(&p) {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("could not find a ready pod")
}

// isPodReady returns true if a pod is ready; false otherwise.
func isPodReady(pod *v1.Pod) bool {
	return isPodReadyConditionTrue(pod.Status)
}

// isPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func isPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := getPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// getPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func getPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := getPodCondition(&status, v1.PodReady)
	return condition
}

// getPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}
