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

package cluster

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// KubeClientFactory returns a Kubernetes client.
type KubeClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error)
}

// ClientFactory returns a Kubernetes client.
type ClientFactory struct {
	clusters          Store
	kubeClientFactory KubeClientFactory
}

// NewClientFactory returns a new ClientFactory.
func NewClientFactory(clusters Store, kubeClientFactory KubeClientFactory) ClientFactory {
	return ClientFactory{
		clusters:          clusters,
		kubeClientFactory: kubeClientFactory,
	}
}

// FromClusterID creates a Kubernetes client for a cluster from a cluster ID.
func (f ClientFactory) FromClusterID(ctx context.Context, clusterID uint) (kubernetes.Interface, error) {
	cluster, err := f.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	return f.kubeClientFactory.FromSecret(ctx, cluster.ConfigSecretID.String())
}

// FromSecret creates a Kubernetes client for a cluster from a secret.
func (f ClientFactory) FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error) {
	return f.kubeClientFactory.FromSecret(ctx, secretID)
}

// DynamicKubeClientFactory returns a dynamic Kubernetes client.
type DynamicKubeClientFactory interface {
	// FromSecret creates a dynamic Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (dynamic.Interface, error)
}

// DynamicClientFactory returns a Kubernetes client.
type DynamicClientFactory struct {
	clusters                 Store
	dynamicKubeClientFactory DynamicKubeClientFactory
}

// NewDynamicClientFactory returns a new DynamicClientFactory.
func NewDynamicClientFactory(clusters Store, dynamicKubeClientFactory DynamicKubeClientFactory) DynamicClientFactory {
	return DynamicClientFactory{
		clusters:                 clusters,
		dynamicKubeClientFactory: dynamicKubeClientFactory,
	}
}

// FromClusterID creates a dynamic Kubernetes client for a cluster from a cluster ID.
func (f DynamicClientFactory) FromClusterID(ctx context.Context, clusterID uint) (dynamic.Interface, error) {
	cluster, err := f.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	return f.dynamicKubeClientFactory.FromSecret(ctx, cluster.ConfigSecretID.String())
}

// FromSecret creates a dynamic Kubernetes client for a cluster from a secret.
func (f DynamicClientFactory) FromSecret(ctx context.Context, secretID string) (dynamic.Interface, error) {
	return f.dynamicKubeClientFactory.FromSecret(ctx, secretID)
}
