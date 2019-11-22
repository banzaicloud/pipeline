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

package clusteradapter

import (
	"context"
	"encoding/base64"
	"fmt"

	"emperror.dev/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	banzaihelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// ClientFactory returns a Kubernetes client.
type ClientFactory struct {
	secretStore common.SecretStore
}

// NewClientFactory returns a new ClientFactory.
func NewClientFactory(secretStore common.SecretStore) ClientFactory {
	return ClientFactory{
		secretStore: secretStore,
	}
}

// FromSecret creates a Kubernetes client for a cluster from a secret.
func (f ClientFactory) FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error) {
	values, err := f.secretStore.GetSecretValues(ctx, secretID)
	if err != nil {
		return nil, err
	}

	// TODO: better secret parsing?
	kubeConfig, err := base64.StdEncoding.DecodeString(values[secrettype.K8SConfig])
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode Kubernetes config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// HelmClientFactory returns a Kubernetes client.
type HelmClientFactory struct {
	secretStore common.SecretStore

	logger common.Logger
}

// NewHelmClientFactory returns a new HelmClientFactory.
func NewHelmClientFactory(secretStore common.SecretStore, logger common.Logger) HelmClientFactory {
	return HelmClientFactory{
		secretStore: secretStore,

		logger: logger,
	}
}

// FromSecret creates a Kubernetes client for a cluster from a secret.
func (f HelmClientFactory) FromSecret(ctx context.Context, secretID string) (*banzaihelm.Client, error) {
	values, err := f.secretStore.GetSecretValues(ctx, secretID)
	if err != nil {
		return nil, err
	}

	// TODO: better secret parsing?
	kubeConfig, err := base64.StdEncoding.DecodeString(values[secrettype.K8SConfig])
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode Kubernetes config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
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
	secretStore common.SecretStore
}

// NewDynamicFileClientFactory returns a new DynamicFileClientFactory.
func NewDynamicFileClientFactory(secretStore common.SecretStore) DynamicFileClientFactory {
	return DynamicFileClientFactory{
		secretStore: secretStore,
	}
}

// FromSecret creates a DynamicFileClient for a cluster from a secret.
func (f DynamicFileClientFactory) FromSecret(ctx context.Context, secretID string) (cluster.DynamicFileClient, error) {
	values, err := f.secretStore.GetSecretValues(ctx, secretID)
	if err != nil {
		return nil, err
	}

	// TODO: better secret parsing?
	kubeConfig, err := base64.StdEncoding.DecodeString(values[secrettype.K8SConfig])
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode Kubernetes config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}

	return k8sclient.NewDynamicFileClient(runtimeClient), nil
}
