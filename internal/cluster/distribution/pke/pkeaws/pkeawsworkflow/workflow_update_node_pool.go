// Copyright © 2020 Banzai Cloud
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

package pkeawsworkflow

import (
	"fmt"

	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
)

const UpdateNodePoolWorkflowName = "pke-aws-update-node-pool"

type UpdateNodePoolWorkflowInput struct {
	ProviderSecretID string
	Region           string

	StackName string

	OrganizationID  uint
	ClusterID       uint
	ClusterSecretID string
	ClusterName     string
	NodePoolName    string

	NodeImage string

	Version string

	Options pke.NodePoolUpdateOptions

	ClusterTags map[string]string
}

type sentinel string

func (e sentinel) Error() string {
	return string(e)
}

func (e sentinel) ServiceError() bool {
	return true
}

const notPipelineEnterpriseError = sentinel("pke nodepool update is supported only in Pipeline Enterprise")

type UpdateNodePoolWorkflow struct{}

// NewUpdateNodePoolWorkflow returns a new UpdateNodePoolWorkflow.
func NewUpdateNodePoolWorkflow() UpdateNodePoolWorkflow {
	return UpdateNodePoolWorkflow{}
}

func (w UpdateNodePoolWorkflow) Register() {
	workflow.RegisterWithOptions(w.Execute, workflow.RegisterOptions{Name: UpdateNodePoolWorkflowName})
}

func (w UpdateNodePoolWorkflow) Execute(ctx workflow.Context, input UpdateNodePoolWorkflowInput) (string, error) {
	statusMessage := fmt.Sprintf("failed to update node pool: %s", notPipelineEnterpriseError.Error())

	_ = SetClusterStatus(ctx, input.ClusterID, cluster.Warning, statusMessage)

	return "", notPipelineEnterpriseError
}
