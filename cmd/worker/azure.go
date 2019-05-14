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
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	azurepkeworkflow "github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

func registerAzureWorkflows(secretStore pkeworkflow.SecretStore, tokenGenerator *pkeworkflowadapter.TokenGenerator, store pke.AzurePKEClusterStore) {

	// Azure PKE
	workflow.RegisterWithOptions(azurepkeworkflow.CreateClusterWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.CreateClusterWorkflowName})
	workflow.RegisterWithOptions(azurepkeworkflow.CreateInfrastructureWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.CreateInfraWorkflowName})
	workflow.RegisterWithOptions(azurepkeworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.DeleteClusterWorkflowName})
	workflow.RegisterWithOptions(azurepkeworkflow.DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.DeleteInfraWorkflowName})
	workflow.RegisterWithOptions(azurepkeworkflow.UpdateClusterWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.UpdateClusterWorkflowName})

	azureClientFactory := azurepkeworkflow.NewAzureClientFactory(secretStore)

	createVnetActivity := azurepkeworkflow.MakeCreateVnetActivity(azureClientFactory)
	activity.RegisterWithOptions(createVnetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateVnetActivityName})

	createNSGActivity := azurepkeworkflow.MakeCreateNSGActivity(azureClientFactory)
	activity.RegisterWithOptions(createNSGActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateNSGActivityName})

	createLBActivity := azurepkeworkflow.MakeCreateLoadBalancerActivity(azureClientFactory)
	activity.RegisterWithOptions(createLBActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateLoadBalancerActivityName})

	createVMSSActivity := azurepkeworkflow.MakeCreateVMSSActivity(azureClientFactory, tokenGenerator)
	activity.RegisterWithOptions(createVMSSActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateVMSSActivityName})

	createRouteTableActivity := azurepkeworkflow.MakeCreateRouteTableActivity(azureClientFactory)
	activity.RegisterWithOptions(createRouteTableActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateRouteTableActivityName})

	assignRoleActivity := azurepkeworkflow.MakeAssignRoleActivity(azureClientFactory)
	activity.RegisterWithOptions(assignRoleActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.AssignRoleActivityName})

	createPublicIPActivity := azurepkeworkflow.MakeCreatePublicIPActivity(azureClientFactory)
	activity.RegisterWithOptions(createPublicIPActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreatePublicIPActivityName})

	// delete infra activities
	deleteVMSSActivity := azurepkeworkflow.MakeDeleteVMSSActivity(azureClientFactory)
	activity.RegisterWithOptions(deleteVMSSActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteVMSSActivityName})

	deleteLoadBalancerActivity := azurepkeworkflow.MakeDeleteLoadBalancerActivity(azureClientFactory)
	activity.RegisterWithOptions(deleteLoadBalancerActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteLoadBalancerActivityName})

	deletePublicIPActivity := azurepkeworkflow.MakeDeletePublicIPActivity(azureClientFactory)
	activity.RegisterWithOptions(deletePublicIPActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeletePublicIPActivityName})

	deleteVNetActivity := azurepkeworkflow.MakeDeleteVNetActivity(azureClientFactory)
	activity.RegisterWithOptions(deleteVNetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteVNetActivityName})

	deleteRouteTableActivity := azurepkeworkflow.MakeDeleteRouteTableActivity(azureClientFactory)
	activity.RegisterWithOptions(deleteRouteTableActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteRouteTableActivityName})

	deleteNSGActivity := azurepkeworkflow.MakeDeleteNSGActivity(azureClientFactory)
	activity.RegisterWithOptions(deleteNSGActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteNSGActivityName})

	deleteClusterFromStoreActivity := azurepkeworkflow.MakeDeleteClusterFromStoreActivity(store)
	activity.RegisterWithOptions(deleteClusterFromStoreActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.DeleteClusterFromStoreActivityName})

	setClusterStatusActivity := azurepkeworkflow.MakeSetClusterStatusActivity(store)
	activity.RegisterWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.SetClusterStatusActivityName})
}
