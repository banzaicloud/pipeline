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
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	vsphereworkflow "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
)

func registerVsphereWorkflows(secretStore pkeworkflow.SecretStore, tokenGenerator pkeworkflowadapter.TokenGenerator) {
	workflow.RegisterWithOptions(vsphereworkflow.CreateClusterWorkflow, workflow.RegisterOptions{Name: vsphereworkflow.CreateClusterWorkflowName})

	vsphereClientFactory := vsphereworkflow.NewVMOMIClientFactory(secretStore)

	createNodeActivity := vsphereworkflow.MakeCreateNodeActivity(vsphereClientFactory, tokenGenerator)
	activity.RegisterWithOptions(createNodeActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.CreateNodeActivityName})
}
