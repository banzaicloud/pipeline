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
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	cluster2 "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/adapter"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	eksworkflow2 "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	"github.com/banzaicloud/pipeline/pkg/sdk/cadence/lib/pipeline/processlog"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func registerEKSWorkflows(
	config configuration,
	secretStore eksworkflow.SecretStore,
	clusterManager *adapter.ClusterManagerAdapter,
	nodePoolStore eks.NodePoolStore,
	clusterDynamicClientFactory cluster2.DynamicClientFactory,
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

	workflow.RegisterWithOptions(cluster.EKSCreateClusterWorkflow, workflow.RegisterOptions{Name: cluster.EKSCreateClusterWorkflowName})

	createInfrastructureWorkflow := eksworkflow.NewCreateInfrastructureWorkflow(nodePoolStore)
	workflow.RegisterWithOptions(createInfrastructureWorkflow.Execute, workflow.RegisterOptions{Name: eksworkflow.CreateInfraWorkflowName})

	awsSessionFactory := eksworkflow.NewAWSSessionFactory(secretStore)
	cloudFormationFactory := eksworkflow.NewCloudFormationFactory()
	ec2Factory := eksworkflow.NewEC2Factory()
	eksFactory := eksworkflow.NewEKSFactory()

	validateRoleNameActivity := eksworkflow.NewValidateIAMRoleActivity(awsSessionFactory)
	activity.RegisterWithOptions(validateRoleNameActivity.Execute, activity.RegisterOptions{Name: eksworkflow.ValidateIAMRoleActivityName})

	createVPCActivity := eksworkflow.NewCreateVPCActivity(awsSessionFactory, vpcTemplate)
	activity.RegisterWithOptions(createVPCActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateVpcActivityName})

	createSubnetActivity := eksworkflow.NewCreateSubnetActivity(awsSessionFactory, subnetTemplate)
	activity.RegisterWithOptions(createSubnetActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateSubnetActivityName})

	getSubnetsDetailsActivity := eksworkflow.NewGetSubnetsDetailsActivity(awsSessionFactory)
	activity.RegisterWithOptions(getSubnetsDetailsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetSubnetsDetailsActivityName})

	createIamRolesActivity := eksworkflow.NewCreateIamRolesActivity(awsSessionFactory, iamRolesTemplate)
	activity.RegisterWithOptions(createIamRolesActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateIamRolesActivityName})

	uploadSSHActivityActivity := eksworkflow.NewUploadSSHKeyActivity(awsSessionFactory)
	activity.RegisterWithOptions(uploadSSHActivityActivity.Execute, activity.RegisterOptions{Name: eksworkflow.UploadSSHKeyActivityName})

	getVpcConfigActivity := eksworkflow.NewGetVpcConfigActivity(awsSessionFactory)
	activity.RegisterWithOptions(getVpcConfigActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetVpcConfigActivityName})

	createEksClusterActivity := eksworkflow.NewCreateEksClusterActivity(awsSessionFactory)
	activity.RegisterWithOptions(createEksClusterActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateEksControlPlaneActivityName})

	saveClusterVersionActivity := eksworkflow.NewSaveClusterVersionActivity(clusterManager)
	activity.RegisterWithOptions(saveClusterVersionActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveClusterVersionActivityName})

	createAsgActivity := eksworkflow.NewCreateAsgActivity(awsSessionFactory, nodePoolTemplate, nodePoolStore)
	activity.RegisterWithOptions(createAsgActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateAsgActivityName})

	updateAsgActivity := eksworkflow.NewUpdateAsgActivity(awsSessionFactory, nodePoolTemplate)
	activity.RegisterWithOptions(updateAsgActivity.Execute, activity.RegisterOptions{Name: eksworkflow.UpdateAsgActivityName})

	createUserAccessKeyActivity := eksworkflow.NewCreateClusterUserAccessKeyActivity(awsSessionFactory)
	activity.RegisterWithOptions(createUserAccessKeyActivity.Execute, activity.RegisterOptions{Name: eksworkflow.CreateClusterUserAccessKeyActivityName})

	bootstrapActivity := eksworkflow.NewBootstrapActivity(awsSessionFactory)
	activity.RegisterWithOptions(bootstrapActivity.Execute, activity.RegisterOptions{Name: eksworkflow.BootstrapActivityName})

	saveK8sConfigActivity := eksworkflow.NewSaveK8sConfigActivity(awsSessionFactory, clusterManager)
	activity.RegisterWithOptions(saveK8sConfigActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveK8sConfigActivityName})

	// update cluster workflow
	eksUpdateClusterWorkflow := cluster.NewEKSUpdateClusterWorkflow(nodePoolStore)
	workflow.RegisterWithOptions(eksUpdateClusterWorkflow.Execute, workflow.RegisterOptions{Name: cluster.EKSUpdateClusterWorkflowName})

	// delete cluster workflow
	workflow.RegisterWithOptions(cluster.EKSDeleteClusterWorkflow, workflow.RegisterOptions{Name: cluster.EKSDeleteClusterWorkflowName})
	workflow.RegisterWithOptions(eksworkflow.DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: eksworkflow.DeleteInfraWorkflowName})

	// delete node pool workflow
	eksworkflow.NewDeleteNodePoolWorkflow().Register()

	getAMISizeActivity := eksworkflow.NewGetAMISizeActivity(awsSessionFactory, ec2Factory)
	activity.RegisterWithOptions(getAMISizeActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetAMISizeActivityName})

	getCFStackActivity := eksworkflow.NewGetCFStackActivity(awsSessionFactory, cloudFormationFactory)
	activity.RegisterWithOptions(getCFStackActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetCFStackActivityName})

	getOwnedELBsActivity := eksworkflow.NewGetOwnedELBsActivity(awsSessionFactory)
	activity.RegisterWithOptions(getOwnedELBsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetOwnedELBsActivityName})

	waitELBsDeletionActivity := eksworkflow.NewWaitELBsDeletionActivity(awsSessionFactory)
	activity.RegisterWithOptions(waitELBsDeletionActivity.Execute, activity.RegisterOptions{Name: eksworkflow.WaitELBsDeletionActivityName})

	deleteNodePoolLabelSetActivity := eksworkflow.NewDeleteNodePoolLabelSetActivity(
		clusterDynamicClientFactory,
		config.Cluster.Labels.Namespace,
	)
	activity.RegisterWithOptions(
		deleteNodePoolLabelSetActivity.Execute,
		activity.RegisterOptions{
			Name: eksworkflow.DeleteNodePoolLabelSetActivityName,
		},
	)

	deleteStackActivity := eksworkflow.NewDeleteStackActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteStackActivityName})

	deleteStoredNodePoolActivity := eksworkflow.NewDeleteStoredNodePoolActivity(nodePoolStore)
	activity.RegisterWithOptions(deleteStoredNodePoolActivity.Execute, activity.RegisterOptions{
		Name: eksworkflow.DeleteStoredNodePoolActivityName,
	})

	deleteControlPlaneActivity := eksworkflow.NewDeleteControlPlaneActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteControlPlaneActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteControlPlaneActivityName})

	deleteSshKeyActivity := eksworkflow.NewDeleteSshKeyActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteSshKeyActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteSshKeyActivityName})

	getOrphanNicsActivity := eksworkflow.NewGetOrphanNICsActivity(awsSessionFactory)
	activity.RegisterWithOptions(getOrphanNicsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetOrphanNICsActivityName})

	listStoredNodePoolsActivity := eksworkflow.NewListStoredNodePoolsActivity(nodePoolStore)
	activity.RegisterWithOptions(listStoredNodePoolsActivity.Execute, activity.RegisterOptions{
		Name: eksworkflow.ListStoredNodePoolsActivityName,
	})

	deleteOrphanNicActivity := eksworkflow.NewDeleteOrphanNICActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteOrphanNicActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteOrphanNICActivityName})

	getSubnetStacksActivity := eksworkflow.NewGetSubnetStacksActivity(awsSessionFactory)
	activity.RegisterWithOptions(getSubnetStacksActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetSubnetStacksActivityName})

	selectVolumeSizeActivity := eksworkflow.NewSelectVolumeSizeActivity(config.Distribution.EKS.DefaultNodeVolumeSize)
	activity.RegisterWithOptions(selectVolumeSizeActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SelectVolumeSizeActivityName})

	setClusterStatusActivity := eksworkflow.NewSetClusterStatusActivity(clusterManager)
	activity.RegisterWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SetClusterStatusActivityName})

	deleteClusterFromStoreActivity := eksworkflow.NewDeleteClusterFromStoreActivity(clusterManager)
	activity.RegisterWithOptions(deleteClusterFromStoreActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteClusterFromStoreActivityName})

	saveNetworkDetails := eksworkflow.NewSaveNetworkDetailsActivity(clusterManager)
	activity.RegisterWithOptions(saveNetworkDetails.Execute, activity.RegisterOptions{Name: eksworkflow.SaveNetworkDetailsActivityName})

	saveNodePoolsActivity := eksworkflow.NewSaveNodePoolsActivity(clusterManager)
	activity.RegisterWithOptions(saveNodePoolsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveNodePoolsActivityName})

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
