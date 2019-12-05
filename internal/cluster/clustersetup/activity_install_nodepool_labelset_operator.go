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

	"emperror.dev/errors"
	"github.com/ghodss/yaml"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterconfig"
)

const InstallNodePoolLabelSetOperatorActivityName = "install-nodepool-labelset-operator"

type InstallNodePoolLabelSetOperatorActivity struct {
	config clusterconfig.LabelConfig

	helmService HelmService
}

// NewInstallNodePoolLabelSetOperatorActivity returns a new InstallNodePoolLabelSetOperatorActivity.
func NewInstallNodePoolLabelSetOperatorActivity(
	config clusterconfig.LabelConfig,
	helmService HelmService,
) InstallNodePoolLabelSetOperatorActivity {
	return InstallNodePoolLabelSetOperatorActivity{
		config:      config,
		helmService: helmService,
	}
}

type InstallNodePoolLabelSetOperatorActivityInput struct {
	ClusterID uint
}

func (a InstallNodePoolLabelSetOperatorActivity) Execute(ctx context.Context, input InstallNodePoolLabelSetOperatorActivityInput) error {
	var config struct {
		Configuration struct {
			// Labeler configuration
			Labeler struct {
				// ForbiddenLabelDomains holds the forbidden domain names, the labeler won't set matching labels
				ForbiddenLabelDomains []string `mapstructure:"forbiddenLabelDomains"`
			} `mapstructure:"labeler"`
		} `json:"configuration,omitempty"`
	}

	config.Configuration.Labeler.ForbiddenLabelDomains = a.config.ForbiddenDomains

	values, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal NodePoolLabelSet operator config to yaml values")
	}

	err = a.helmService.InstallDeployment(
		ctx,
		input.ClusterID,
		a.config.Namespace,
		a.config.Charts.NodepoolLabelOperator.Chart,
		"npls",
		values,
		a.config.Charts.NodepoolLabelOperator.Version,
		true,
	)

	if err != nil {
		return errors.WithMessage(err, "installing NodePoolLabelSet operator failed")
	}

	return nil
}
