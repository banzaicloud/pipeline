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

// +testify:mock:testOnly=true

// ClientFactory returns a Kubernetes client.
type ClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error)
}

// +testify:mock:testOnly=true

// DynamicClientFactory returns a dynamic Kubernetes client.
type DynamicClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (dynamic.Interface, error)
}

// +testify:mock:testOnly=true

// HelmClientFactory returns a Kubernetes client.
type HelmClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (*helm.Client, error)
}

// +testify:mock:testOnly=true

// DynamicFileClientFactory returns a DynamicFileClient.
type DynamicFileClientFactory interface {
	// FromSecret creates a DynamicFileClient for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (k8s.DynamicFileClient, error)
}
