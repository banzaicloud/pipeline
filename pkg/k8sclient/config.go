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
	"emperror.dev/emperror"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClientConfig creates a Kubernetes client config from raw kube config.
func NewClientConfig(kubeConfig []byte) (*rest.Config, error) {
	if kubeConfig == nil {
		return nil, errors.New("kube config is empty")
	}
	apiconfig, err := clientcmd.Load(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to load kubernetes API config")
	}

	apiconfig, err = cleanConfig(apiconfig)
	if err != nil {
		return nil, err
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to build client config from API config")
	}

	return config, nil
}
