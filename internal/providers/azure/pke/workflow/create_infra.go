package workflow

import "go.uber.org/cadence/workflow"

const CreateInfraWorkflowName = "pke-azure-create-infra"

type CreateAzureInfrastructureWorkflowInput struct {
}

func CreateInfrastructureWorkflow(ctx workflow.Context, input CreateAzureInfrastructureWorkflowInput) error {
	// Create VNET

	// Create Subnet

	// CreateNetworkSecurity Group

	// Create BasicLoadbalancer

	// Create ScaleSet

	// Set AssignRolePolicy
	return nil
}
