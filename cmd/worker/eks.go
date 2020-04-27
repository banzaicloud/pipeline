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

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/adapter"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	eksworkflow2 "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func registerEKSWorkflows(secretStore eksworkflow.SecretStore, clusterManager *adapter.ClusterManagerAdapter) error {
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
	workflow.RegisterWithOptions(eksworkflow.CreateInfrastructureWorkflow, workflow.RegisterOptions{Name: eksworkflow.CreateInfraWorkflowName})

	awsSessionFactory := eksworkflow.NewAWSSessionFactory(secretStore)

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

	createAsgActivity := eksworkflow.NewCreateAsgActivity(awsSessionFactory, nodePoolTemplate)
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
	workflow.RegisterWithOptions(cluster.EKSUpdateClusterWorkflow, workflow.RegisterOptions{Name: cluster.EKSUpdateClusterWorkflowName})

	// delete cluster workflow
	workflow.RegisterWithOptions(cluster.EKSDeleteClusterWorkflow, workflow.RegisterOptions{Name: cluster.EKSDeleteClusterWorkflowName})
	workflow.RegisterWithOptions(eksworkflow.DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: eksworkflow.DeleteInfraWorkflowName})

	getOwnedELBsActivity := eksworkflow.NewGetOwnedELBsActivity(awsSessionFactory)
	activity.RegisterWithOptions(getOwnedELBsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetOwnedELBsActivityName})

	waitELBsDeletionActivity := eksworkflow.NewWaitELBsDeletionActivity(awsSessionFactory)
	activity.RegisterWithOptions(waitELBsDeletionActivity.Execute, activity.RegisterOptions{Name: eksworkflow.WaitELBsDeletionActivityName})

	getNodepoolStacksActivity := eksworkflow.NewGetNodepoolStacksActivity(awsSessionFactory)
	activity.RegisterWithOptions(getNodepoolStacksActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetNodepoolStacksActivityName})

	deleteStackActivity := eksworkflow.NewDeleteStackActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteStackActivityName})

	deleteControlPlaneActivity := eksworkflow.NewDeleteControlPlaneActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteControlPlaneActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteControlPlaneActivityName})

	deleteSshKeyActivity := eksworkflow.NewDeleteSshKeyActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteSshKeyActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteSshKeyActivityName})

	getOrphanNicsActivity := eksworkflow.NewGetOrphanNICsActivity(awsSessionFactory)
	activity.RegisterWithOptions(getOrphanNicsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetOrphanNICsActivityName})

	deleteOrphanNicActivity := eksworkflow.NewDeleteOrphanNICActivity(awsSessionFactory)
	activity.RegisterWithOptions(deleteOrphanNicActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteOrphanNICActivityName})

	getSubnetStacksActivity := eksworkflow.NewGetSubnetStacksActivity(awsSessionFactory)
	activity.RegisterWithOptions(getSubnetStacksActivity.Execute, activity.RegisterOptions{Name: eksworkflow.GetSubnetStacksActivityName})

	setClusterStatusActivity := eksworkflow.NewSetClusterStatusActivity(clusterManager)
	activity.RegisterWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SetClusterStatusActivityName})

	deleteClusterFromStoreActivity := eksworkflow.NewDeleteClusterFromStoreActivity(clusterManager)
	activity.RegisterWithOptions(deleteClusterFromStoreActivity.Execute, activity.RegisterOptions{Name: eksworkflow.DeleteClusterFromStoreActivityName})

	saveNetworkDetails := eksworkflow.NewSaveNetworkDetailsActivity(clusterManager)
	activity.RegisterWithOptions(saveNetworkDetails.Execute, activity.RegisterOptions{Name: eksworkflow.SaveNetworkDetailsActivityName})

	saveNodePoolsActivity := eksworkflow.NewSaveNodePoolsActivity(clusterManager)
	activity.RegisterWithOptions(saveNodePoolsActivity.Execute, activity.RegisterOptions{Name: eksworkflow.SaveNodePoolsActivityName})

	// Node pool upgrade
	workflow.RegisterWithOptions(eksworkflow2.UpdateNodePoolWorkflow, workflow.RegisterOptions{Name: eksworkflow2.UpdateNodePoolWorkflowName})

	updateNodeGroupActivity := eksworkflow2.NewUpdateNodeGroupActivity(awsSessionFactory, nodePoolTemplate)
	activity.RegisterWithOptions(updateNodeGroupActivity.Execute, activity.RegisterOptions{Name: eksworkflow2.UpdateNodeGroupActivityName})

	waitCloudFormationStackUpdateActivity := eksworkflow2.NewWaitCloudFormationStackUpdateActivity(awsSessionFactory)
	activity.RegisterWithOptions(waitCloudFormationStackUpdateActivity.Execute, activity.RegisterOptions{Name: eksworkflow2.WaitCloudFormationStackUpdateActivityName})

	return nil
}
