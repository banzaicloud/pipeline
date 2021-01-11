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

package cluster

import (
	"context"

	"go.uber.org/cadence/workflow"
)

const InstallIntServiceOperatorWorkflowName = "install-int-service-operator"

type InstallIntServiceOperatorInput struct {
	ClusterID uint
}

type InstallIntegratedServiceOperatorWorkflow struct {
	helmService HelmService
}

func NewInstallIntegratedServiceOperatorWorkflow(helmService HelmService) *InstallIntegratedServiceOperatorWorkflow {
	return &InstallIntegratedServiceOperatorWorkflow{
		helmService: helmService,
	}
}

func (w *InstallIntegratedServiceOperatorWorkflow) Execute(ctx workflow.Context, input InstallIntServiceOperatorInput) error {
	logger := workflow.GetLogger(ctx).Sugar().With(
		"clusterID", input.ClusterID,
	)

	logger.Info("start installing integrated service operator")

	return w.helmService.InstallDeployment(context.Background(), input.ClusterID, "pipeline-system", "banzaicloud-stable/integrated-service-operator", "iso", nil, "", false)
}
