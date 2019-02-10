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
	"github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
	"k8s.io/api/core/v1"
)
import "github.com/banzaicloud/pipeline/config"

type nodePoolLabelSetOperatorConfig struct {
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	Affinity    v1.Affinity     `json:"affinity,omitempty"`
}

// InstallNodePoolLabelSetOperator deploys node pool label set operator.
func InstallNodePoolLabelSetOperator(cluster CommonCluster) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	headNodeAffinity := getHeadNodeAffinity(cluster)
	headNodeTolerations := getHeadNodeTolerations()

	chartName := helm.BanzaiRepository + "/nodepool-labels-operator"
	chartVersion := viper.GetString(config.NodePoolLabelSetOperatorChartVersion)

	config := nodePoolLabelSetOperatorConfig{
		Tolerations: headNodeTolerations,
		Affinity:    headNodeAffinity,
	}

	overrideValues, err := yaml.Marshal(config)
	if err != nil {
		return emperror.Wrap(err, "failed to marshal NodePoolLabelSet operator config to yaml values")
	}

	err = installDeployment(
		cluster,
		pipelineSystemNamespace,
		chartName,
		"npls",
		overrideValues,
		chartVersion,
		true,
	)

	if err != nil {
		return emperror.Wrap(err, "installing node pool labelset operator failed")
	}

	return nil
}
