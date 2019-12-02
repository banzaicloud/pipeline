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

package kubernetes

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/common"
	banzaihelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// ClientFactory returns a Kubernetes client.
type ClientFactory struct {
	configFactory ConfigFactory
}

// NewClientFactory returns a new ClientFactory.
func NewClientFactory(configFactory ConfigFactory) ClientFactory {
	return ClientFactory{
		configFactory: configFactory,
	}
}

// FromSecret creates a Kubernetes client for a cluster from a secret.
func (f ClientFactory) FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error) {
	config, err := f.configFactory.FromSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	return k8sclient.NewClientFromConfig(config)
}

// HelmClientFactory returns a Kubernetes client.
type HelmClientFactory struct {
	configFactory ConfigFactory

	logger common.Logger
}

// NewHelmClientFactory returns a new HelmClientFactory.
func NewHelmClientFactory(configFactory ConfigFactory, logger common.Logger) HelmClientFactory {
	return HelmClientFactory{
		configFactory: configFactory,

		logger: logger,
	}
}

// FromSecret creates a Kubernetes client for a cluster from a secret.
func (f HelmClientFactory) FromSecret(ctx context.Context, secretID string) (*banzaihelm.Client, error) {
	config, err := f.configFactory.FromSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	client, err := k8sclient.NewClientFromConfig(config)
	if err != nil {
		return nil, err
	}

	f.logger.Debug("create kubernetes tunnel")
	tillerTunnel, err := portforwarder.New("kube-system", client, config)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create kubernetes tunnel")
	}

	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	f.logger.Debug("created kubernetes tunnel on address", map[string]interface{}{"address": tillerTunnelAddress})

	hClient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return &banzaihelm.Client{Tunnel: tillerTunnel, Client: hClient}, nil
}

// DynamicFileClientFactory returns a DynamicFileClient.
type DynamicFileClientFactory struct {
	configFactory ConfigFactory
}

// NewDynamicFileClientFactory returns a new DynamicFileClientFactory.
func NewDynamicFileClientFactory(configFactory ConfigFactory) DynamicFileClientFactory {
	return DynamicFileClientFactory{
		configFactory: configFactory,
	}
}

// FromSecret creates a DynamicFileClient for a cluster from a secret.
func (f DynamicFileClientFactory) FromSecret(ctx context.Context, secretID string) (cluster.DynamicFileClient, error) {
	config, err := f.configFactory.FromSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}

	return k8sclient.NewDynamicFileClient(runtimeClient), nil
}
