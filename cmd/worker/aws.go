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

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
)

func registerAwsWorkflows(
	worker worker.Worker,
	config configuration,
	clusters *pkeworkflowadapter.ClusterManagerAdapter,
	tokenGenerator pkeworkflowadapter.TokenGenerator,
	secretStore pkeworkflow.SecretStore,
	imageSelector pkeaws.ImageSelector,
	awsSecretStore awsworkflow.SecretStore,
) {
	createClusterWorkflow := pkeworkflow.NewCreateClusterWorkflow(
		config.Distribution.PKE.Amazon.DefaultNodeVolumeSize,
		config.Distribution.PKE.Amazon.GlobalRegion,
	)
	worker.RegisterWorkflowWithOptions(
		createClusterWorkflow.Execute, workflow.RegisterOptions{Name: pkeworkflow.CreateClusterWorkflowName},
	)

	worker.RegisterWorkflowWithOptions(pkeworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: pkeworkflow.DeleteClusterWorkflowName})

	updateClusterWorkflow := pkeworkflow.UpdateClusterWorkflow{
		DefaultNodeVolumeSize: config.Distribution.PKE.Amazon.DefaultNodeVolumeSize,
	}
	worker.RegisterWorkflowWithOptions(
		updateClusterWorkflow.Execute, workflow.RegisterOptions{Name: pkeworkflow.UpdateClusterWorkflowName},
	)

	awsClientFactory := pkeworkflow.NewAWSClientFactory(secretStore)
	ec2Factory := pkeworkflow.NewEC2Factory()
	elbv2Factory := pkeawsworkflow.NewELBV2Factory()

	createAWSRolesActivity := pkeworkflow.NewCreateAWSRolesActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(createAWSRolesActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateAWSRolesActivityName})

	waitCFCompletionActivity := pkeworkflow.NewWaitCFCompletionActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(waitCFCompletionActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.WaitCFCompletionActivityName})

	createPKEVPCActivity := pkeworkflow.NewCreateVPCActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(createPKEVPCActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateVPCActivityName})

	createPKESubnetActivity := pkeworkflow.NewCreateSubnetActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(createPKESubnetActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateSubnetActivityName})

	deletePKESubnetActivity := pkeworkflow.NewDeleteSubnetActivity(clusters)
	worker.RegisterActivityWithOptions(deletePKESubnetActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteSubnetActivityName})

	getVpcDefaultSecurityGroupActivity := pkeworkflow.NewGetVpcDefaultSecurityGroupActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(getVpcDefaultSecurityGroupActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.GetVpcDefaultSecurityGroupActivityName})

	updateClusterStatusActivitiy := pkeworkflow.NewUpdateClusterStatusActivity(clusters)
	worker.RegisterActivityWithOptions(updateClusterStatusActivitiy.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdateClusterStatusActivityName})

	updateClusterNetworkActivitiy := pkeworkflow.NewUpdateClusterNetworkActivity(clusters)
	worker.RegisterActivityWithOptions(updateClusterNetworkActivitiy.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdateClusterNetworkActivityName})

	createElasticIPActivity := pkeworkflow.NewCreateElasticIPActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(createElasticIPActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateElasticIPActivityName})

	createNLBActivity := pkeworkflow.NewCreateNLBActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(createNLBActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateNLBActivityName})

	createMasterActivity := pkeworkflow.NewCreateMasterActivity(clusters, tokenGenerator)
	worker.RegisterActivityWithOptions(createMasterActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateMasterActivityName})

	listNodePoolsActivity := pkeworkflow.NewListNodePoolsActivity(clusters)
	worker.RegisterActivityWithOptions(listNodePoolsActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.ListNodePoolsActivityName})

	selectImageActivity := pkeworkflow.NewSelectImageActivity(clusters, imageSelector)
	worker.RegisterActivityWithOptions(selectImageActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.SelectImageActivityName})

	createWorkerPoolActivity := pkeworkflow.NewCreateWorkerPoolActivity(clusters, tokenGenerator)
	worker.RegisterActivityWithOptions(createWorkerPoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateWorkerPoolActivityName})

	pkeworkflow.NewUpdateNodeGroupActivity(awsClientFactory, clusters, tokenGenerator, config.Pipeline.External.URL, config.Pipeline.External.Insecure).Register(worker)

	pkeworkflow.NewUpdateMasterNodeGroupActivity(awsClientFactory, clusters, tokenGenerator, config.Pipeline.External.URL, config.Pipeline.External.Insecure).Register(worker)

	pkeworkflow.NewCalculateNodePoolVersionActivity().Register(worker)

	deletePoolActivity := pkeworkflow.NewDeletePoolActivity(clusters)
	worker.RegisterActivityWithOptions(deletePoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeletePoolActivityName})

	waitForDeletePoolActivity := pkeworkflow.NewWaitForDeletePoolActivity(clusters)
	worker.RegisterActivityWithOptions(waitForDeletePoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.WaitForDeletePoolActivityName})

	updatePoolActivity := pkeworkflow.NewUpdatePoolActivity(awsClientFactory)
	worker.RegisterActivityWithOptions(updatePoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdatePoolActivityName})

	deleteElasticIPActivity := pkeworkflow.NewDeleteElasticIPActivity(clusters)
	worker.RegisterActivityWithOptions(deleteElasticIPActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteElasticIPActivityName})

	deleteNLBActivity := pkeworkflow.NewDeleteNLBActivity(clusters)
	worker.RegisterActivityWithOptions(deleteNLBActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteNLBActivityName})

	waitForDeleteNLBActivity := pkeworkflow.NewWaitForDeleteNLBActivity(clusters)
	worker.RegisterActivityWithOptions(waitForDeleteNLBActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.WaitForDeleteNLBActivityName})

	deleteVPCActivity := pkeworkflow.NewDeleteVPCActivity(clusters)
	worker.RegisterActivityWithOptions(deleteVPCActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteVPCActivityName})

	waitForDeleteVPCActivity := pkeworkflow.NewWaitForDeleteVPCActivity(clusters)
	worker.RegisterActivityWithOptions(waitForDeleteVPCActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.WaitForDeleteVPCActivityName})

	uploadSshKeyPairActivity := pkeworkflow.NewUploadSSHKeyPairActivity(clusters)
	worker.RegisterActivityWithOptions(uploadSshKeyPairActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.UploadSSHKeyPairActivityName})

	deleteSshKeyPairActivity := pkeworkflow.NewDeleteSSHKeyPairActivity(clusters)
	worker.RegisterActivityWithOptions(deleteSshKeyPairActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteSSHKeyPairActivityName})

	selectVolumeSizeActivity := pkeworkflow.NewSelectVolumeSizeActivity(awsClientFactory, ec2Factory)
	worker.RegisterActivityWithOptions(selectVolumeSizeActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.SelectVolumeSizeActivityName})

	pkeawsworkflow.NewHealthCheckActivity(awsClientFactory, elbv2Factory).Register(worker)

	awsSessionFactory := awsworkflow.NewAWSSessionFactory(awsSecretStore)
	deleteStackActivity := awsworkflow.NewDeleteStackActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: awsworkflow.DeleteStackActivityName})
}
