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

	"go.uber.org/cadence/workflow"
)

const CreateClusterWorkflowName = "eks-create-cluster"

// CreateClusterWorkflowInput holds data needed by the create cluster workflow
type CreateClusterWorkflowInput struct {
	CreateInfrastructureWorkflowInput
	ClusterID uint
}

type CreateClusterWorkflowOutput struct {
	CreateInfrastructureWorkflowOutput
}

// CreateClusterWorkflow executes the Cadence workflow responsible for creating and configuring an EKS cluster
func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) (*CreateClusterWorkflowOutput, error) {
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 1 * time.Hour,
		TaskStartToCloseTimeout:      5 * time.Minute,
	}
	ctx = workflow.WithChildOptions(ctx, cwo)

	// create infra child workflow
	infraInput := CreateInfrastructureWorkflowInput{
		ClusterUID:            input.ClusterUID,
		Region:                input.Region,
		OrganizationID:        input.OrganizationID,
		SecretID:              input.SecretID,
		ClusterName:           input.ClusterName,
		DefaultUser:           input.DefaultUser,
		VpcCidr:               input.VpcCidr,
		VpcID:                 input.VpcID,
		RouteTableID:          input.RouteTableID,
		Subnets:               input.Subnets,
		SSHSecretID:           input.SSHSecretID,
		AsgList:               input.AsgList,
		LogTypes:              input.LogTypes,
		KubernetesVersion:     input.KubernetesVersion,
		ASGSubnetMapping:      input.ASGSubnetMapping,
		ClusterRoleID:         input.ClusterRoleID,
		NodeInstanceRoleID:    input.NodeInstanceRoleID,
		EndpointPublicAccess:  input.EndpointPublicAccess,
		EndpointPrivateAccess: input.EndpointPublicAccess,
	}

	infraOutput := CreateInfrastructureWorkflowOutput{}
	err := workflow.ExecuteChildWorkflow(ctx, CreateInfraWorkflowName, infraInput).Get(ctx, &infraOutput)
	if err != nil {
		return nil, err
	}

	return &CreateClusterWorkflowOutput{
		CreateInfrastructureWorkflowOutput: infraOutput}, nil

}
