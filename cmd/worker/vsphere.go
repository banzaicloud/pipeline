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

	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	vsphereworkflow "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/secret/kubesecret"
)

func registerVsphereWorkflows(worker worker.Worker, secretStore pkeworkflow.SecretStore, tokenGenerator pkeworkflowadapter.TokenGenerator, store pke.ClusterStore, kubeSecretStore kubesecret.KubeSecretStore) {
	worker.RegisterWorkflowWithOptions(vsphereworkflow.NewCreateClusterWorkflow().Execute, workflow.RegisterOptions{Name: vsphereworkflow.CreateClusterWorkflowName})

	vsphereClientFactory := vsphereworkflow.NewVMOMIClientFactory(secretStore)

	createNodeActivity := vsphereworkflow.MakeCreateNodeActivity(vsphereClientFactory, tokenGenerator, secretStore)
	worker.RegisterActivityWithOptions(createNodeActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.CreateNodeActivityName})

	waitForIpActivity := vsphereworkflow.MakeWaitForIPActivity(vsphereClientFactory)
	worker.RegisterActivityWithOptions(waitForIpActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.WaitForIPActivityName})

	setClusterStatusActivity := vsphereworkflow.MakeSetClusterStatusActivity(store)
	worker.RegisterActivityWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.SetClusterStatusActivityName})

	worker.RegisterWorkflowWithOptions(vsphereworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: vsphereworkflow.DeleteClusterWorkflowName})

	deleteNodeActivity := vsphereworkflow.MakeDeleteNodeActivity(vsphereClientFactory)
	worker.RegisterActivityWithOptions(deleteNodeActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.DeleteNodeActivityName})

	deleteClusterFromStoreActivity := vsphereworkflow.MakeDeleteClusterFromStoreActivity(store)
	worker.RegisterActivityWithOptions(deleteClusterFromStoreActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.DeleteClusterFromStoreActivityName})

	worker.RegisterWorkflowWithOptions(vsphereworkflow.UpdateClusterWorkflow, workflow.RegisterOptions{Name: vsphereworkflow.UpdateClusterWorkflowName})

	worker.RegisterWorkflowWithOptions(vsphereworkflow.DeleteNodePoolWorkflow, workflow.RegisterOptions{Name: vsphereworkflow.DeleteNodePoolWorkflowName})

	getPublicAddressActivity := vsphereworkflow.MakeGetPublicAddressActivity(vsphereClientFactory)
	worker.RegisterActivityWithOptions(getPublicAddressActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.GetPublicAddressActivityName})

	deleteK8sNodeActivity := vsphereworkflow.MakeDeleteK8sNodeActivity(kubeSecretStore)
	worker.RegisterActivityWithOptions(deleteK8sNodeActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.DeleteK8sNodeActivityName})

	deleteNodePoolFromStoreActivity := vsphereworkflow.MakeDeleteNodePoolFromStoreActivity(store)
	worker.RegisterActivityWithOptions(deleteNodePoolFromStoreActivity.Execute, activity.RegisterOptions{Name: vsphereworkflow.DeleteNodePoolFromStoreActivityName})
}
