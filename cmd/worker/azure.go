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
	azurepkeworkflow "github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

func registerAzureWorkflows(secretStore pkeworkflow.SecretStore) {

	// Azure PKE
	workflow.RegisterWithOptions(azurepkeworkflow.CreateClusterWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.CreateClusterWorkflowName})
	workflow.RegisterWithOptions(azurepkeworkflow.CreateInfrastructureWorkflow, workflow.RegisterOptions{Name: azurepkeworkflow.CreateInfraWorkflowName})

	azureClientFactory := azurepkeworkflow.NewAzureClientFactory(secretStore)

	createVnetActivity := azurepkeworkflow.MakeCreateVnetActivity(azureClientFactory)
	activity.RegisterWithOptions(createVnetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateVnetActivityName})

	createSubnetActivity := azurepkeworkflow.MakeCreateSubnetActivity(azureClientFactory)
	activity.RegisterWithOptions(createSubnetActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateSubnetActivityName})

	createNSGActivity := azurepkeworkflow.MakeCreateNSGActivity(azureClientFactory)
	activity.RegisterWithOptions(createNSGActivity.Execute, activity.RegisterOptions{Name: azurepkeworkflow.CreateNSGActivityName})
}
