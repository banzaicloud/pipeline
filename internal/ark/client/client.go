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

package client

import (
	clientset "github.com/heptio/ark/pkg/generated/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientService is an interface for a implementation which gives back an initialized ARK client
type ClientService interface {
	GetClient() (*Client, error)
}

// Client describes an ARK client
type Client struct {
	Config    []byte
	Logger    logrus.FieldLogger
	Namespace string

	Client clientset.Interface
}

// New creates an initialized Client instance
func New(config []byte, namespace string, logger logrus.FieldLogger) (client *Client, err error) {

	client = &Client{
		Config:    config,
		Logger:    logger,
		Namespace: namespace,
	}

	client.Client, err = client.new(config)
	if err != nil {
		return
	}

	return
}

// getK8sClientConfig creates a Kubernetes client config
func (c *Client) getK8sClientConfig(kubeConfig []byte) (config *rest.Config, err error) {

	if kubeConfig != nil {
		apiconfig, err := clientcmd.Load(kubeConfig)
		if err != nil {
			return config, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return config, err
		}
		c.Logger.Debug("Use K8S RemoteCluster Config: ", config.ServerName)
	} else {
		err = errors.New("kubeconfig value is nil")
		return
	}

	return
}

// new initializes an ARK client from a k8s config
func (c *Client) new(config []byte) (clientset.Interface, error) {

	clientConfig, err := c.getK8sClientConfig(config)
	if err != nil {
		return nil, err
	}

	arkClient, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return arkClient, nil
}
