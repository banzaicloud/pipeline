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
	"emperror.dev/emperror"
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/global"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

type nodePoolLabelSetOperatorConfig struct {
	Tolerations   []v1.Toleration `json:"tolerations,omitempty"`
	Affinity      v1.Affinity     `json:"affinity,omitempty"`
	Configuration configuration   `json:"configuration,omitempty"`
}

type configuration struct {
	// Labeler configuration
	Labeler labelerConfig `mapstructure:"labeler"`
}

type labelerConfig struct {
	// ForbiddenLabelDomains holds the forbidden domain names, the labeler won't set matching labels
	ForbiddenLabelDomains []string `mapstructure:"forbiddenLabelDomains"`
}

// InstallNodePoolLabelSetOperator deploys node pool label set operator.
func InstallNodePoolLabelSetOperator(cluster CommonCluster) error {
	pipelineSystemNamespace := global.Config.Cluster.Labels.Namespace
	reservedNodeLabelDomains := global.Config.Cluster.Labels.ForbiddenDomains

	headNodeAffinity := GetHeadNodeAffinity(cluster)
	headNodeTolerations := GetHeadNodeTolerations()

	chartName := global.Config.Cluster.Labels.Charts.NodepoolLabelOperator.Chart
	chartVersion := global.Config.Cluster.Labels.Charts.NodepoolLabelOperator.Version

	config := nodePoolLabelSetOperatorConfig{
		Tolerations: headNodeTolerations,
		Affinity:    headNodeAffinity,
		Configuration: configuration{
			Labeler: labelerConfig{
				ForbiddenLabelDomains: reservedNodeLabelDomains,
			},
		},
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
		return emperror.Wrap(err, "installing NodePoolLabelSet operator failed")
	}

	return nil
}

type NodePoolLabelParam struct {
	Labels map[string]map[string]string `json:"labels"`
}

// SetupNodePoolLabelsSet deploys NodePoolLabelSet resources for each nodepool.
func SetupNodePoolLabelsSet(cluster CommonCluster, param pkgCluster.PostHookParam) error {
	var nodePoolParam NodePoolLabelParam
	err := castToPostHookParam(&param, &nodePoolParam)
	if err != nil {
		return emperror.Wrap(err, "posthook param failed")
	}

	return DeployNodePoolLabelsSet(cluster, nodePoolParam.Labels)
}
