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
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"

	cluster2 "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/adapter"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	eksworkflow2 "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/sdk/cadence/lib/pipeline/processlog"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func registerEKSWorkflows(
	worker worker.Worker,
	config configuration,
	secretStore awsworkflow.SecretStore,
	clusterManager *adapter.ClusterManagerAdapter,
	nodePoolStore eks.NodePoolStore,
	clusterDynamicClientFactory cluster2.DynamicClientFactory,
	database *gorm.DB,
) error {
	vpcTemplate, err := eksworkflow.GetVPCTemplate()
	if err != nil {
		return errors.WrapIf(err, "failed to get CloudFormation template for VPC")
	}

	subnetTemplate, err := eksworkflow.GetSubnetTemplate()
	if err != nil {
		return errors.WrapIf(err, "failed to get CloudFormation template for Subnet")
	}

	iamRolesTemplate, err := eksworkflow.GetIAMTemplate()
	if err != nil {
		return errors.WrapIf(err, "failed to get CloudFormation template for IAM roles")
	}

	nodePoolTemplate, err := eksworkflow.GetNodePoolTemplate()
	if err != nil {
		return errors.WrapIf(err, "failed to get CloudFormation template for node pools")
	}

	// Activities.
	eksworkflow.NewCreateStoredNodePoolActivity(nodePoolStore).Register()
	eksworkflow.NewListStoredEKSClustersActivity(database).Register()
	eksworkflow.NewSetNodePoolStatusActivity(nodePoolStore).Register()

	// Workflows.
	eksworkflow.NewCreateNodePoolWorkflow().Register()

	worker.RegisterWorkflowWithOptions(cluster.EKSCreateClusterWorkflow, workflow.RegisterOptions{Name: cluster.EKSCreateClusterWorkflowName})

	createInfrastructureWorkflow := eksworkflow.NewCreateInfrastructureWorkflow(nodePoolStore)
	worker.RegisterWorkflowWithOptions(createInfrastructureWorkflow.Execute, workflow.RegisterOptions{Name: eksworkflow.CreateInfraWorkflowName})

	awsSessionFactory := awsworkflow.NewAWSSessionFactory(secretStore)
	cloudFormationFactory := awsworkflow.NewCloudFormationFactory()
	ec2Factory := eksworkflow.NewEC2Factory()
	eksFactory := eksworkflow.NewEKSFactory()

	validateRoleNameActivity := eksworkflow.NewValidateIAMRoleActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(validateRoleNameActivity.Execute, activity.RegisterOptions{Name: eksworkflow.ValidateIAMRoleActivityName})

	createVPCActivity := eksworkflow.NewCreateVPCActivity(awsSessionFactory, vpcTemplate)
	worker.RegisterActivityWithOptions(createVPCActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateVpcActivityName})

	createSubnetActivity := eksworkflow.NewCreateSubnetActivity(awsSessionFactory, subnetTemplate)
	worker.RegisterActivityWithOptions(createSubnetActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateSubnetActivityName})

	getSubnetsDetailsActivity := eksworkflow.NewGetSubnetsDetailsActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(getSubnetsDetailsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetSubnetsDetailsActivityName})

	createIamRolesActivity := eksworkflow.NewCreateIamRolesActivity(awsSessionFactory, iamRolesTemplate)
	worker.RegisterActivityWithOptions(createIamRolesActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateIamRolesActivityName})

	uploadSSHActivityActivity := eksworkflow.NewUploadSSHKeyActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(uploadSSHActivityActivity.Execute, activity.RegisterOptions{Name: eksworkflow.UploadSSHKeyActivityName})

	getVpcConfigActivity := eksworkflow.NewGetVpcConfigActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(getVpcConfigActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetVpcConfigActivityName})

	createEksClusterActivity := eksworkflow.NewCreateEksClusterActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(createEksClusterActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateEksControlPlaneActivityName})

	saveClusterVersionActivity := eksworkflow.NewSaveClusterVersionActivity(clusterManager)
	worker.RegisterActivityWithOptions(saveClusterVersionActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveClusterVersionActivityName})

	createAsgActivity := eksworkflow.NewCreateAsgActivity(awsSessionFactory, nodePoolTemplate, nodePoolStore)
	worker.RegisterActivityWithOptions(createAsgActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateAsgActivityName})

	updateAsgActivity := eksworkflow.NewUpdateAsgActivity(awsSessionFactory, nodePoolTemplate)
	worker.RegisterActivityWithOptions(updateAsgActivity.Execute, activity.RegisterOptions{Name: eksworkflow.UpdateAsgActivityName})

	createUserAccessKeyActivity := eksworkflow.NewCreateClusterUserAccessKeyActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(createUserAccessKeyActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateClusterUserAccessKeyActivityName})

	bootstrapActivity := eksworkflow.NewBootstrapActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(bootstrapActivity.Execute, activity.RegisterOptions{Name: eksworkflow.BootstrapActivityName})

	saveK8sConfigActivity := eksworkflow.NewSaveK8sConfigActivity(awsSessionFactory, clusterManager)
	worker.RegisterActivityWithOptions(saveK8sConfigActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveK8sConfigActivityName})

	// update cluster workflow
	eksUpdateClusterWorkflow := cluster.NewEKSUpdateClusterWorkflow(nodePoolStore)
	worker.RegisterWorkflowWithOptions(eksUpdateClusterWorkflow.Execute, workflow.RegisterOptions{Name: cluster.EKSUpdateClusterWorkflowName})

	// delete cluster workflow
	worker.RegisterWorkflowWithOptions(cluster.EKSDeleteClusterWorkflow, workflow.RegisterOptions{Name: cluster.EKSDeleteClusterWorkflowName})
	worker.RegisterWorkflowWithOptions(eksworkflow.DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: eksworkflow.DeleteInfraWorkflowName})

	// delete node pool workflow
	eksworkflow.NewDeleteNodePoolWorkflow().Register()

	getAMISizeActivity := eksworkflow.NewGetAMISizeActivity(awsSessionFactory, ec2Factory)
	worker.RegisterActivityWithOptions(getAMISizeActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetAMISizeActivityName})

	getCFStackActivity := eksworkflow.NewGetCFStackActivity(awsSessionFactory, cloudFormationFactory)
	worker.RegisterActivityWithOptions(getCFStackActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetCFStackActivityName})

	getOwnedELBsActivity := eksworkflow.NewGetOwnedELBsActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(getOwnedELBsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetOwnedELBsActivityName})

	waitELBsDeletionActivity := eksworkflow.NewWaitELBsDeletionActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(waitELBsDeletionActivity.Execute, activity.RegisterOptions{Name: eksworkflow.WaitELBsDeletionActivityName})

	deleteStoredNodePoolActivity := eksworkflow.NewDeleteStoredNodePoolActivity(nodePoolStore)
	worker.RegisterActivityWithOptions(deleteStoredNodePoolActivity.Execute, activity.RegisterOptions{
		Name: eksworkflow.DeleteStoredNodePoolActivityName,
	})

	deleteControlPlaneActivity := eksworkflow.NewDeleteControlPlaneActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(deleteControlPlaneActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteControlPlaneActivityName})

	deleteSshKeyActivity := eksworkflow.NewDeleteSshKeyActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(deleteSshKeyActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteSshKeyActivityName})

	getOrphanNicsActivity := eksworkflow.NewGetOrphanNICsActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(getOrphanNicsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetOrphanNICsActivityName})

	listStoredNodePoolsActivity := eksworkflow.NewListStoredNodePoolsActivity(nodePoolStore)
	worker.RegisterActivityWithOptions(listStoredNodePoolsActivity.Execute, activity.RegisterOptions{
		Name: eksworkflow.ListStoredNodePoolsActivityName,
	})

	deleteOrphanNicActivity := eksworkflow.NewDeleteOrphanNICActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(deleteOrphanNicActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteOrphanNICActivityName})

	getSubnetStacksActivity := eksworkflow.NewGetSubnetStacksActivity(awsSessionFactory)
	worker.RegisterActivityWithOptions(getSubnetStacksActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetSubnetStacksActivityName})

	selectVolumeSizeActivity := eksworkflow.NewSelectVolumeSizeActivity(config.Distribution.EKS.DefaultNodeVolumeSize)
	worker.RegisterActivityWithOptions(selectVolumeSizeActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SelectVolumeSizeActivityName})

	setClusterStatusActivity := eksworkflow.NewSetClusterStatusActivity(clusterManager)
	worker.RegisterActivityWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SetClusterStatusActivityName})

	deleteClusterFromStoreActivity := eksworkflow.NewDeleteClusterFromStoreActivity(clusterManager)
	worker.RegisterActivityWithOptions(deleteClusterFromStoreActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteClusterFromStoreActivityName})

	saveNetworkDetails := eksworkflow.NewSaveNetworkDetailsActivity(clusterManager)
	worker.RegisterActivityWithOptions(saveNetworkDetails.Execute, activity.RegisterOptions{Name: eksworkflow.SaveNetworkDetailsActivityName})

	saveNodePoolsActivity := eksworkflow.NewSaveNodePoolsActivity(clusterManager)
	worker.RegisterActivityWithOptions(saveNodePoolsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveNodePoolsActivityName})

	// Node pool upgrade
	eksworkflow2.NewUpdateNodePoolWorkflow(awsSessionFactory, cloudFormationFactory, processlog.New()).Register()

	eksworkflow2.NewCalculateNodePoolVersionActivity().Register()
	eksworkflow2.NewUpdateNodeGroupActivity(awsSessionFactory, nodePoolTemplate).Register()
	eksworkflow2.NewWaitCloudFormationStackUpdateActivity(awsSessionFactory).Register()

	// New cluster update
	eksworkflow2.NewUpdateClusterWorkflow().Register()

	eksworkflow2.NewUpdateClusterVersionActivity(awsSessionFactory, eksFactory).Register()
	eksworkflow2.NewWaitUpdateClusterVersionActivity(awsSessionFactory, eksFactory).Register()

	return nil
}
