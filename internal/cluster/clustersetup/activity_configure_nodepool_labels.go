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

	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
)

const ConfigureNodePoolLabelsActivityName = "configure-nodepool-labels"

type ConfigureNodePoolLabelsActivity struct {
	namespace string

	clientFactory DynamicClientFactory
}

// NewConfigureNodePoolLabelsActivity returns a new ConfigureNodePoolLabelsActivity.
func NewConfigureNodePoolLabelsActivity(
	namespace string,
	clientFactory DynamicClientFactory,
) ConfigureNodePoolLabelsActivity {
	return ConfigureNodePoolLabelsActivity{
		namespace:     namespace,
		clientFactory: clientFactory,
	}
}

type ConfigureNodePoolLabelsActivityInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string

	Labels map[string]map[string]string
}

func (a ConfigureNodePoolLabelsActivity) Execute(ctx context.Context, input ConfigureNodePoolLabelsActivityInput) error {
	client, err := a.clientFactory.FromSecret(ctx, input.ConfigSecretID)
	if err != nil {
		return err
	}

	manager := npls.NewManager(client, a.namespace)

	return manager.Sync(input.Labels)
}
