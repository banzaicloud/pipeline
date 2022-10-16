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
	"github.com/Masterminds/semver/v3"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterconfig"
	"github.com/banzaicloud/pipeline/internal/global"
)

const DeployAWSEBSCSIDriverActivityName = "deploy-aws-ebs-csi-driver"

type DeployAWSEBSCSIDriverActivity struct {
	config      clusterconfig.LabelConfig
	helmService HelmService
}

type DeployAWSEBSCSIDriverActivityInput struct {
	ClusterID         uint
	KubernetesVersion string
	ChartVersion      string
}

// NewDeployAWSEBSCSIDriverActivity returns a new DeployAWSEBSCSIDriverActivity.
func NewDeployAWSEBSCSIDriverActivity(
	config clusterconfig.LabelConfig,
	helmService HelmService,
) DeployAWSEBSCSIDriverActivity {
	return DeployAWSEBSCSIDriverActivity{
		config:      config,
		helmService: helmService,
	}
}

func (a DeployAWSEBSCSIDriverActivity) Execute(ctx context.Context, input DeployAWSEBSCSIDriverActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"clusterID", input.ClusterID,
		"KubernetesVersion", input.KubernetesVersion,
		"chartVersion", input.ChartVersion,
	)

	ebsCSIDriverConstraint, err := semver.NewConstraint(">= 1.23")
	if err != nil {
		return errors.WrapIf(err, "creating semver constraint for EBS CSI driver failed")
	}

	k8sVersion, err := semver.NewVersion(input.KubernetesVersion)
	if err != nil {
		return errors.WrapIf(err, "creating semver from Kubernetes version failed")
	}

	if !ebsCSIDriverConstraint.Check(k8sVersion) {
		logger.Infof("kubernetesVersion failed ebsCSIDriverConstraint check", "k8sVersion", k8sVersion)

		return nil
	}

	return a.helmService.ApplyDeployment(
		ctx,
		input.ClusterID,
		global.Config.Cluster.Namespace,
		"aws-ebs-csi-driver/aws-ebs-csi-driver",
		"aws-ebs-csi-driver",
		nil,
		input.ChartVersion,
	)
}
