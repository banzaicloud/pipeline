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

package main

import (
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	azurepkeworkflow "github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
)

func registerAzureWorkflows(worker worker.Worker, secretStore pkeworkflow.SecretStore, tokenGenerator pkeworkflowadapter.TokenGenerator, store pke.ClusterStore) {
	// Azure PKE
	worker.RegisterWorkflowWithOptions(azurepkeworkflow.NewCreateClusterWorkflow().Execute, workflow.RegisterOptions{Name: azurepkeworkflow.CreateClusterWorkflowName})
	worker.RegisterWorkflowWithOptions(azurepkeworkflow.CreateInfrastructureWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.CreateInfraWorkflowName})
	worker.RegisterWorkflowWithOptions(azurepkeworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.DeleteClusterWorkflowName})
	worker.RegisterWorkflowWithOptions(azurepkeworkflow.DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.DeleteInfraWorkflowName})
	worker.RegisterWorkflowWithOptions(azurepkeworkflow.UpdateClusterWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.UpdateClusterWorkflowName})

	azureClientFactory := azurepkeworkflow.NewAzureClientFactory(secretStore)

	createVnetActivity := azurepkeworkflow.MakeCreateVnetActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(createVnetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateVnetActivityName})

	createNSGActivity := azurepkeworkflow.MakeCreateNSGActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(createNSGActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateNSGActivityName})

	createLBActivity := azurepkeworkflow.MakeCreateLoadBalancerActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(createLBActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateLoadBalancerActivityName})

	createVMSSActivity := azurepkeworkflow.MakeCreateVMSSActivity(azureClientFactory, tokenGenerator)
	worker.RegisterActivityWithOptions(createVMSSActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateVMSSActivityName})

	createRouteTableActivity := azurepkeworkflow.MakeCreateRouteTableActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(createRouteTableActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateRouteTableActivityName})

	assignRoleActivity := azurepkeworkflow.MakeAssignRoleActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(assignRoleActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.AssignRoleActivityName})

	createPublicIPActivity := azurepkeworkflow.MakeCreatePublicIPActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(createPublicIPActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreatePublicIPActivityName})

	// delete infra activities
	deleteVMSSActivity := azurepkeworkflow.MakeDeleteVMSSActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deleteVMSSActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteVMSSActivityName})

	deleteLoadBalancerActivity := azurepkeworkflow.MakeDeleteLoadBalancerActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deleteLoadBalancerActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteLoadBalancerActivityName})

	deletePublicIPActivity := azurepkeworkflow.MakeDeletePublicIPActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deletePublicIPActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeletePublicIPActivityName})

	deleteVNetActivity := azurepkeworkflow.MakeDeleteVNetActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deleteVNetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteVNetActivityName})

	deleteRouteTableActivity := azurepkeworkflow.MakeDeleteRouteTableActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deleteRouteTableActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteRouteTableActivityName})

	deleteNSGActivity := azurepkeworkflow.MakeDeleteNSGActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deleteNSGActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteNSGActivityName})

	deleteClusterFromStoreActivity := azurepkeworkflow.MakeDeleteClusterFromStoreActivity(store)
	worker.RegisterActivityWithOptions(deleteClusterFromStoreActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteClusterFromStoreActivityName})

	setClusterStatusActivity := azurepkeworkflow.MakeSetClusterStatusActivity(store)
	worker.RegisterActivityWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.SetClusterStatusActivityName})

	updateVMSSActivity := azurepkeworkflow.MakeUpdateVMSSActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(updateVMSSActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.UpdateVMSSActivityName})

	createSubnetActivity := azurepkeworkflow.MakeCreateSubnetActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(createSubnetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateSubnetActivityName})

	deleteNodePoolFromStoreActivity := azurepkeworkflow.MakeDeleteNodePoolFromStoreActivity(store)
	worker.RegisterActivityWithOptions(deleteNodePoolFromStoreActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteNodePoolFromStoreActivityName})

	deleteSubnetActivity := azurepkeworkflow.MakeDeleteSubnetActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(deleteSubnetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteSubnetActivityName})

	collectUpdateClusterProvidersActivity := azurepkeworkflow.MakeCollectUpdateClusterProvidersActivity(azureClientFactory)
	worker.RegisterActivityWithOptions(collectUpdateClusterProvidersActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CollectUpdateClusterProvidersActivityName})

	updateClusterAccessPointsActivity := azurepkeworkflow.MakeUpdateClusterAccessPointsActivity(store)
	worker.RegisterActivityWithOptions(updateClusterAccessPointsActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.UpdateClusterAccessPointsActivityName})
}
