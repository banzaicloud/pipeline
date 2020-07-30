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

package testing

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func KubeConfigFromEnv(t *testing.T) []byte {
	kubeConfigFile := os.Getenv("KUBECONFIG")
	if kubeConfigFile == "" {
		t.Skip("skipping as Kubernetes config was not provided")
	}
	kubeConfigBytes, err := ioutil.ReadFile(kubeConfigFile)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	return kubeConfigBytes
}

func EnsureNamespaceRemoved(client *kubernetes.Clientset, namespace string, timeout time.Duration) error {
	ctx := context.Background()

	nsList, err := client.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ns := range nsList.Items {
		if ns.Name == namespace {
			err := client.CoreV1().Namespaces().Delete(ctx, namespace, v1.DeleteOptions{})
			if err != nil {
				return err
			}
			err = wait.Poll(time.Second, timeout, func() (done bool, err error) {
				nsList, err := client.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
				if err != nil {
					return false, err
				}
				for _, ns := range nsList.Items {
					if ns.Name == namespace {
						return false, nil
					}
				}
				return true, nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
