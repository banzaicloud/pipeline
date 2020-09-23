// Copyright © 2029 Banzai Cloud
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

package workflow

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// +testify:mock:testOnly=true

// ClientFactory returns a Kubernetes client.
type ClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error)
}

// NewSimpleClientFactory returns a new ClientFactory that always returns the same clientset.
// It is mostly useful in simple unit tests.
func NewSimpleClientFactory(clientset kubernetes.Interface) ClientFactory {
	return simpleClientFactory{
		clientset: clientset,
	}
}

type simpleClientFactory struct {
	clientset kubernetes.Interface
}

func (f simpleClientFactory) FromSecret(_ context.Context, _ string) (kubernetes.Interface, error) {
	return f.clientset, nil
}
