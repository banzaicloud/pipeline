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

package helm

import (
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
		"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
)

var tillerTunnel *kube.Tunnel

// GetK8sConnection creates a new Kubernetes client.
// Deprecated: use github.com/banzaicloud/pipeline/pkg/k8sclient.NewClientFromKubeConfig
func GetK8sConnection(kubeConfig []byte) (*kubernetes.Clientset, error) {
	return k8sclient.NewClientFromKubeConfig(kubeConfig)
}

// GetK8sInClusterConnection returns Kubernetes in-cluster configuration.
// Deprecated: use github.com/banzaicloud/pipeline/pkg/k8sclient.NewInClusterClient
func GetK8sInClusterConnection() (*kubernetes.Clientset, error) {
	return k8sclient.NewInClusterClient()
}

// GetK8sClientConfig creates a Kubernetes client config.
// Deprecated: use github.com/banzaicloud/pipeline/pkg/k8sclient.NewClientConfig
func GetK8sClientConfig(kubeConfig []byte) (*rest.Config, error) {
	return k8sclient.NewClientConfig(kubeConfig)
}

//GetHelmClient establishes Tunnel for Helm client TODO check client and config if both needed
func GetHelmClient(kubeConfig []byte) (*helm.Client, error) {
	for i := 0; i < 2; i++ {
		log.Debug("Create kubernetes Client.")
		config, err := GetK8sClientConfig(kubeConfig)
		if err != nil {
			log.Debug("Could not get K8S config")
			return nil, err
		}

		client, err := GetK8sConnection(kubeConfig)
		if err != nil {
			log.Debug("Could not create kubernetes client from config.")
			return nil, fmt.Errorf("create kubernetes client failed: %v", err)
		}
		log.Debug("Create kubernetes Tunnel")
		tillerTunnel, err = portforwarder.New("kube-system", client, config)
		if err != nil {
			if err.Error() == "Unauthorized" && i == 0 {
				log.Errorf("create tunnel attempt %d/%d failed: %s", i+1, 2, err.Error())
				time.Sleep(time.Millisecond * 20)
				continue
			}
			return nil, fmt.Errorf("create tunnel attempt %d/%d failed: %s", i+1, 2, err.Error())
		}
		break
	}
	log.Debug("Created kubernetes tunnel on address: localhost:", tillerTunnel.Local)
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hclient := helm.NewClient(helm.Host(tillerTunnelAddress))
	return hclient, nil
}

//CheckDeploymentState checks the state of Helm deployment
func CheckDeploymentState(kubeConfig []byte, releaseName string) (string, error) {
	client, err := GetK8sConnection(kubeConfig)
	if err != nil {
		return "", errors.Wrap(err, "Error during getting K8S config")
	}

	filter := fmt.Sprintf("release=%s", releaseName)

	state := v1.PodRunning
	podList, err := client.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: filter})
	if err != nil && podList != nil {
		return "", fmt.Errorf("PoD list failed: %v", err)
	}
	for _, pod := range podList.Items {
		log.Debug("PodStatus:", pod.Status.Phase)
		if pod.Status.Phase == v1.PodRunning {
			continue
		} else {
			state = pod.Status.Phase
			break
		}
	}
	return string(state), nil
}

//CreateNamespaceIfNotExist Create Kubernetes Namespace if not exist.
func CreateNamespaceIfNotExist(kubeConfig []byte, namespace string) error {
	client, err := GetK8sConnection(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "Error during getting K8S config")
	}
	_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if apierrors.IsAlreadyExists(err) {
		log.Debugf("Namespace: %s already exist.", namespace)
		return nil
	} else if err != nil {
		log.Errorf("Failed to create namespace %s: %v", namespace, err)
		return err
	}

	log.Infof("Namespace: %s created.", namespace)
	return nil
}
