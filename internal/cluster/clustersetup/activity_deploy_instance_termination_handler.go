// Copyright © 2021 Banzai Cloud
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
	v1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterconfig"
	"github.com/banzaicloud/pipeline/internal/global"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const DeployInstanceTerminationHandlerActivityName = "deploy-instance-termination-handler"

type DeployInstanceTerminationHandlerActivity struct {
	config      clusterconfig.LabelConfig
	helmService HelmService
}

type DeployInstanceTerminationHandlerActivityInput struct {
	ClusterID   uint
	OrgID       uint
	ClusterName string
	Cloud       string
}

// NewDeployInstanceTerminationHandlerActivity returns a new DeployInstanceTerminationHandlerActivity.
func NewDeployInstanceTerminationHandlerActivity(
	config clusterconfig.LabelConfig,
	helmService HelmService,
) DeployInstanceTerminationHandlerActivity {
	return DeployInstanceTerminationHandlerActivity{
		config:      config,
		helmService: helmService,
	}
}

func (a DeployInstanceTerminationHandlerActivity) Execute(ctx context.Context, input DeployInstanceTerminationHandlerActivityInput) error {
	config := global.Config.Cluster.PostHook.ITH
	if !global.Config.Pipeline.Enterprise || !config.Enabled {
		return nil
	}

	if input.Cloud != pkgCluster.Amazon && input.Cloud != pkgCluster.Google {
		return nil
	}

	pipelineSystemNamespace := global.Config.Cluster.Namespace

	values := map[string]interface{}{
		"tolerations": []v1.Toleration{
			{
				Operator: v1.TolerationOpExists,
			},
		},
	}

	marshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal yaml values")
	}

	return a.helmService.ApplyDeployment(context.Background(), input.ClusterID, pipelineSystemNamespace, config.Chart, "ith", marshalledValues, config.Version)
}
