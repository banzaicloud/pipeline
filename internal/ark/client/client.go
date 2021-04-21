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
	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/sirupsen/logrus"
	arkAPI "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

func init() {
	_ = arkAPI.AddToScheme(scheme.Scheme)
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = clientgoscheme.AddToScheme(scheme.Scheme)
}

// ClientService is an interface for a implementation which gives back an initialized ARK client
type ClientService interface {
	GetClient() (*Client, error)
}

// Client describes an ARK client
type Client struct {
	Config    []byte
	Logger    logrus.FieldLogger
	Namespace string

	Client runtimeclient.Client
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

// new initializes an ARK client from a k8s config
func (c *Client) new(kubeconfig []byte) (runtimeclient.Client, error) {
	config, err := k8sclient.NewClientConfig(kubeconfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not create rest config from kubeconfig")
	}

	client, err := runtimeclient.New(config, runtimeclient.Options{})
	if err != nil {
		return nil, errors.WrapIf(err, "could not create runtime client")
	}

	return client, nil
}
