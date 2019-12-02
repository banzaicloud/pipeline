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
	"encoding/base64"

	"emperror.dev/errors"
	"github.com/banzaicloud/nodepool-labels-operator/pkg/npls"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const ConfigureNodePoolLabelsActivityName = "configure-nodepool-labels"

type ConfigureNodePoolLabelsActivity struct {
	namespace string

	secretStore   common.SecretStore
	clientFactory ClientFactory
}

// NewConfigureNodePoolLabelsActivity returns a new ConfigureNodePoolLabelsActivity.
func NewConfigureNodePoolLabelsActivity(
	namespace string,
	secretStore common.SecretStore,
	clientFactory ClientFactory,
) ConfigureNodePoolLabelsActivity {
	return ConfigureNodePoolLabelsActivity{
		namespace:     namespace,
		secretStore:   secretStore,
		clientFactory: clientFactory,
	}
}

type ConfigureNodePoolLabelsActivityInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string

	Labels map[string]map[string]string
}

func (a ConfigureNodePoolLabelsActivity) Execute(ctx context.Context, input ConfigureNodePoolLabelsActivityInput) error {
	desiredLabels := make(npls.NodepoolLabelSets)

	for name, nodePoolLabelMap := range input.Labels {
		if len(nodePoolLabelMap) > 0 {
			desiredLabels[name] = nodePoolLabelMap
		}
	}

	// TODO: drop this once npls can work with a runtime client
	values, err := a.secretStore.GetSecretValues(ctx, input.ConfigSecretID)
	if err != nil {
		return err
	}

	// TODO: better secret parsing?
	kubeConfig, err := base64.StdEncoding.DecodeString(values[secrettype.K8SConfig])
	if err != nil {
		return errors.Wrap(err, "cannot decode Kubernetes config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "cannot create Kubernetes config")
	}

	/*client, err := a.clientFactory.FromSecret(ctx, input.ConfigSecretID)
	if err != nil {
		return err
	}*/

	manager, err := npls.NewNPLSManager(config, a.namespace)
	if err != nil {
		return err
	}

	err = manager.Sync(desiredLabels)
	if err != nil {
		return err
	}

	return nil
}
