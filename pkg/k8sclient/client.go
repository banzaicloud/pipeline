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

package k8sclient

import (
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewClient creates a new Kubernetes client from config.
func NewClientFromConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create client for config")
	}

	return client, nil
}

// NewClientFromKubeConfig creates a new Kubernetes client from raw kube config.
func NewClientFromKubeConfig(kubeConfig []byte) (*kubernetes.Clientset, error) {
	config, err := NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create client config")
	}

	return NewClientFromConfig(config)
}

// NewInClusterClient returns a Kubernetes client based on in-cluster configuration.
func NewInClusterClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to fetch in-cluster configuration")
	}

	return NewClientFromConfig(config)
}
