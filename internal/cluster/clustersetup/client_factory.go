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

package clustersetup

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/pkg/helm"
	k8s "github.com/banzaicloud/pipeline/pkg/kubernetes"
)

// ClientFactory returns a Kubernetes client.
//go:generate mockery -name ClientFactory -inpkg -testonly
type ClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error)
}

// DynamicClientFactory returns a dynamic Kubernetes client.
//go:generate mockery -name DynamicClientFactory -inpkg -testonly
type DynamicClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (dynamic.Interface, error)
}

// HelmClientFactory returns a Kubernetes client.
//go:generate mockery -name HelmClientFactory -inpkg -testonly
type HelmClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (*helm.Client, error)
}

// DynamicFileClientFactory returns a DynamicFileClient.
//go:generate mockery -name DynamicFileClientFactory -inpkg -testonly
type DynamicFileClientFactory interface {
	// FromSecret creates a DynamicFileClient for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (k8s.DynamicFileClient, error)
}
