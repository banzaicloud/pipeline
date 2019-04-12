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

package workflow

import (
	"time"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"go.uber.org/cadence/workflow"
)

const CreateClusterWorkflowName = "pke-azure-create-cluster"

// CreateClusterWorkflowInput
type CreateClusterWorkflowInput struct {
	OrganizationID     uint
	ClusterID          uint
	ClusterUID         string
	ClusterName        string
	SecretID           string
	Location           string
	ResourceGroupName  string
	VirtualNetworkName string
	NodePools          []pke.NodePool
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {

	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithChildOptions(ctx, cwo)

	// TODO fill input
	infraInput := CreateAzureInfrastructureWorkflowInput{
		OrganizationID:    input.OrganizationID,
		ClusterName:       input.ClusterName,
		SecretID:          input.SecretID,
		ResourceGroupName: input.ResourceGroupName,
	}
	err := workflow.ExecuteChildWorkflow(ctx, CreateInfraWorkflowName, infraInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
