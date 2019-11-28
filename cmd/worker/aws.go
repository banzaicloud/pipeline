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
	"github.com/banzaicloud/pipeline/src/secret"
)

func registerAwsWorkflows(clusters *pkeworkflowadapter.ClusterManagerAdapter, tokenGenerator pkeworkflowadapter.TokenGenerator) {
	workflow.RegisterWithOptions(pkeworkflow.CreateClusterWorkflow, workflow.RegisterOptions{Name: pkeworkflow.CreateClusterWorkflowName})
	workflow.RegisterWithOptions(pkeworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: pkeworkflow.DeleteClusterWorkflowName})
	workflow.RegisterWithOptions(pkeworkflow.UpdateClusterWorkflow, workflow.RegisterOptions{Name: pkeworkflow.UpdateClusterWorkflowName})

	awsClientFactory := pkeworkflow.NewAWSClientFactory(pkeworkflowadapter.NewSecretStore(secret.Store))

	createAWSRolesActivity := pkeworkflow.NewCreateAWSRolesActivity(awsClientFactory)
	activity.RegisterWithOptions(createAWSRolesActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateAWSRolesActivityName})

	waitCFCompletionActivity := pkeworkflow.NewWaitCFCompletionActivity(awsClientFactory)
	activity.RegisterWithOptions(waitCFCompletionActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.WaitCFCompletionActivityName})

	createPKEVPCActivity := pkeworkflow.NewCreateVPCActivity(awsClientFactory)
	activity.RegisterWithOptions(createPKEVPCActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateVPCActivityName})

	getVpcDefaultSecurityGroupActivity := pkeworkflow.NewGetVpcDefaultSecurityGroupActivity(awsClientFactory)
	activity.RegisterWithOptions(getVpcDefaultSecurityGroupActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.GetVpcDefaultSecurityGroupActivityName})

	updateClusterStatusActivitiy := pkeworkflow.NewUpdateClusterStatusActivity(clusters)
	activity.RegisterWithOptions(updateClusterStatusActivitiy.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdateClusterStatusActivityName})

	updateClusterNetworkActivitiy := pkeworkflow.NewUpdateClusterNetworkActivity(clusters)
	activity.RegisterWithOptions(updateClusterNetworkActivitiy.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdateClusterNetworkActivityName})

	createElasticIPActivity := pkeworkflow.NewCreateElasticIPActivity(awsClientFactory)
	activity.RegisterWithOptions(createElasticIPActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateElasticIPActivityName})

	createNLBActivity := pkeworkflow.NewCreateNLBActivity(awsClientFactory)
	activity.RegisterWithOptions(createNLBActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateNLBActivityName})

	createMasterActivity := pkeworkflow.NewCreateMasterActivity(clusters, tokenGenerator)
	activity.RegisterWithOptions(createMasterActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateMasterActivityName})

	listNodePoolsActivity := pkeworkflow.NewListNodePoolsActivity(clusters)
	activity.RegisterWithOptions(listNodePoolsActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.ListNodePoolsActivityName})

	createWorkerPoolActivity := pkeworkflow.NewCreateWorkerPoolActivity(clusters, tokenGenerator)
	activity.RegisterWithOptions(createWorkerPoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateWorkerPoolActivityName})

	deletePoolActivity := pkeworkflow.NewDeletePoolActivity(clusters)
	activity.RegisterWithOptions(deletePoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeletePoolActivityName})

	updatePoolActivity := pkeworkflow.NewUpdatePoolActivity(awsClientFactory)
	activity.RegisterWithOptions(updatePoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdatePoolActivityName})

	deleteElasticIPActivity := pkeworkflow.NewDeleteElasticIPActivity(clusters)
	activity.RegisterWithOptions(deleteElasticIPActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteElasticIPActivityName})

	deleteNLBActivity := pkeworkflow.NewDeleteNLBActivity(clusters)
	activity.RegisterWithOptions(deleteNLBActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteNLBActivityName})

	deleteVPCActivity := pkeworkflow.NewDeleteVPCActivity(clusters)
	activity.RegisterWithOptions(deleteVPCActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteVPCActivityName})

	uploadSshKeyPairActivity := pkeworkflow.NewUploadSSHKeyPairActivity(clusters)
	activity.RegisterWithOptions(uploadSshKeyPairActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.UploadSSHKeyPairActivityName})

	deleteSshKeyPairActivity := pkeworkflow.NewDeleteSSHKeyPairActivity(clusters)
	activity.RegisterWithOptions(deleteSshKeyPairActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteSSHKeyPairActivityName})

}
