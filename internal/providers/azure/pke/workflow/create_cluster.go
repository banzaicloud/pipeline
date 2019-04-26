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

	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"go.uber.org/cadence/workflow"
)

const CreateClusterWorkflowName = "pke-azure-create-cluster"

// CreateClusterWorkflowInput
type CreateClusterWorkflowInput struct {
	ClusterID                       uint
	ClusterName                     string
	OrganizationID                  uint
	ResourceGroupName               string
	SecretID                        string
	VirtualNetworkTemplate          VirtualNetworkTemplate
	LoadBalancerTemplate            LoadBalancerTemplate
	PublicIPAddress                 PublicIPAddress
	RoleAssignmentTemplates         []RoleAssignmentTemplate
	RouteTable                      RouteTable
	SecurityGroups                  []SecurityGroup
	VirtualMachineScaleSetTemplates []VirtualMachineScaleSetTemplate
	PostHooks                       pkgCluster.PostHooks
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {

	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithChildOptions(ctx, cwo)

	infraInput := CreateAzureInfrastructureWorkflowInput{
		OrganizationID:    input.OrganizationID,
		ClusterID:         input.ClusterID,
		ClusterName:       input.ClusterName,
		SecretID:          input.SecretID,
		ResourceGroupName: input.ResourceGroupName,
		LoadBalancer: LoadBalancerFactory{
			Template: input.LoadBalancerTemplate,
		},
		PublicIPAddress: input.PublicIPAddress,
		RoleAssignments: RoleAssignmentsFactory{
			Templates: input.RoleAssignmentTemplates,
		},
		RouteTable: input.RouteTable,
		ScaleSets: VirtualMachineScaleSetsFactory{
			Templates: input.VirtualMachineScaleSetTemplates,
		},
		SecurityGroups: input.SecurityGroups,
		VirtualNetwork: VirtualNetworkFactory{
			Template: input.VirtualNetworkTemplate,
		},
	}
	err := workflow.ExecuteChildWorkflow(ctx, CreateInfraWorkflowName, infraInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	postHookWorkflowInput := cluster.RunPostHooksWorkflowInput{
		ClusterID: input.ClusterID,
		PostHooks: cluster.BuildWorkflowPostHookFunctions(input.PostHooks, true),
	}

	err = workflow.ExecuteChildWorkflow(ctx, cluster.RunPostHooksWorkflowName, postHookWorkflowInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
