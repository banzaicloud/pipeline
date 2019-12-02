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
	"encoding/base64"

	"emperror.dev/errors"
	"k8s.io/client-go/rest"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// ConfigFactory returns a Kubernetes configuration.
type ConfigFactory interface {
	// FromSecret returns a config from a secret.
	FromSecret(ctx context.Context, secretID string) (*rest.Config, error)
}

// DefaultConfigFactory is a default implementation of the ConfigFactory interface.
type DefaultConfigFactory struct {
	secrets ConfigSecretStore
}

type ConfigSecretStore interface {
	// GetSecretValues returns the values stored within a secret.
	// If the underlying store uses additional keys for determining the exact secret path
	// (eg. organization ID), it should be retrieved from the context.
	GetSecretValues(ctx context.Context, secretID string) (map[string]string, error)
}

// NewConfigFactory returns a new ConfigFactory.
func NewConfigFactory(secrets ConfigSecretStore) DefaultConfigFactory {
	return DefaultConfigFactory{
		secrets: secrets,
	}
}

// FromSecret returns a config from a secret.
func (f DefaultConfigFactory) FromSecret(ctx context.Context, secretID string) (*rest.Config, error) {
	values, err := f.secrets.GetSecretValues(ctx, secretID)
	if err != nil {
		return nil, err
	}

	// TODO: better secret parsing?
	kubeConfig, err := base64.StdEncoding.DecodeString(values[secrettype.K8SConfig])
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode Kubernetes config")
	}

	return k8sclient.NewClientConfig(kubeConfig)
}
