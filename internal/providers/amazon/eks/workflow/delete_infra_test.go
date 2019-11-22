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

package workflow

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
)

type DeleteClusterInfraWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestDeleteClusterInfraWorkflowTestSuite(t *testing.T) {
	workflow.RegisterWithOptions(DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: DeleteInfraWorkflowName})

	getVpcConfigActivity := NewGetVpcConfigActivity(nil)
	activity.RegisterWithOptions(getVpcConfigActivity.Execute, activity.RegisterOptions{Name: GetVpcConfigActivityName})

	getOwnedELBsActivity := NewGetOwnedELBsActivity(nil)
	activity.RegisterWithOptions(getOwnedELBsActivity.Execute, activity.RegisterOptions{Name: GetOwnedELBsActivityName})

	waitELBsDeletionActivity := NewWaitELBsDeletionActivity(nil)
	activity.RegisterWithOptions(waitELBsDeletionActivity.Execute, activity.RegisterOptions{Name: WaitELBsDeletionActivityName})

	getNodepoolStacksActivity := NewGetNodepoolStacksActivity(nil)
	activity.RegisterWithOptions(getNodepoolStacksActivity.Execute, activity.RegisterOptions{Name: GetNodepoolStacksActivityName})

	deleteStackActivity := NewDeleteStackActivity(nil, "")
	activity.RegisterWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: DeleteStackActivityName})

	deleteControlPlaneActivity := NewDeleteControlPlaneActivity(nil)
	activity.RegisterWithOptions(deleteControlPlaneActivity.Execute, activity.RegisterOptions{Name: DeleteControlPlaneActivityName})

	deleteSshKeyActivity := NewDeleteSshKeyActivity(nil)
	activity.RegisterWithOptions(deleteSshKeyActivity.Execute, activity.RegisterOptions{Name: DeleteSshKeyActivityName})

	deleteClusterUserAccessKeyActivity := NewDeleteClusterUserAccessKeyActivity(nil)
	activity.RegisterWithOptions(deleteClusterUserAccessKeyActivity.Execute, activity.RegisterOptions{Name: DeleteClusterUserAccessKeyActivityName})

	getOrphanNicsActivity := NewGetOrphanNICsActivity(nil)
	activity.RegisterWithOptions(getOrphanNicsActivity.Execute, activity.RegisterOptions{Name: GetOrphanNICsActivityName})

	deleteOrphanNicActivity := NewDeleteOrphanNICActivity(nil)
	activity.RegisterWithOptions(deleteOrphanNicActivity.Execute, activity.RegisterOptions{Name: DeleteOrphanNICActivityName})

	getSubnetStacksActivity := NewGetSubnetStacksActivity(nil)
	activity.RegisterWithOptions(getSubnetStacksActivity.Execute, activity.RegisterOptions{Name: GetSubnetStacksActivityName})

	suite.Run(t, new(DeleteClusterInfraWorkflowTestSuite))
}

func (s *DeleteClusterInfraWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *DeleteClusterInfraWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *DeleteClusterInfraWorkflowTestSuite) Test_Successful_Delete_Infra() {

	workflowInput := DeleteInfrastructureWorkflowInput{
		Region:         "us-west-1",
		OrganizationID: 1,
		SecretID:       "my-secret-id",
		ClusterName:    "test-cluster-name",
		NodePoolNames:  []string{"pool1", "pool2"},
	}

	eksActivityInput := EKSActivityInput{
		OrganizationID: workflowInput.OrganizationID,
		SecretID:       workflowInput.SecretID,
		Region:         workflowInput.Region,
		ClusterName:    workflowInput.ClusterName,
	}

	s.env.OnActivity(GetVpcConfigActivityName, mock.Anything, GetVpcConfigActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        "pipeline-eks-test-cluster-name",
	}).Return(&GetVpcConfigActivityOutput{
		VpcID:               "test-vpc-id",
		SecurityGroupID:     "test-control-plane-sg-id",
		NodeSecurityGroupID: "test-node-sg-id",
	}, nil)

	s.env.OnActivity(GetOwnedELBsActivityName, mock.Anything, GetOwnedELBsActivityInput{
		EKSActivityInput: eksActivityInput,
		VpcID:            "test-vpc-id",
	}).Return(&GetOwnedELBsActivityOutput{
		LoadBalancerNames: []string{"test-lb-1", "test-lb-2"},
	}, nil)

	s.env.OnActivity(WaitELBsDeletionActivityName, mock.Anything, WaitELBsDeletionActivityActivityInput{
		EKSActivityInput:  eksActivityInput,
		LoadBalancerNames: []string{"test-lb-1", "test-lb-2"},
	}).Return(nil)

	s.env.OnActivity(GetNodepoolStacksActivityName, mock.Anything, GetNodepoolStacksActivityInput{
		EKSActivityInput: eksActivityInput,
		NodePoolNames:    []string{"pool1", "pool2"},
	}).Return(&GetNodepoolStacksActivityOutput{
		StackNames: []string{
			generateNodePoolStackName(eksActivityInput.ClusterName, "pool1"),
			generateNodePoolStackName(eksActivityInput.ClusterName, "pool2"),
		},
	}, nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        generateNodePoolStackName(eksActivityInput.ClusterName, "pool1"),
	}).Return(nil).Once()
	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        generateNodePoolStackName(eksActivityInput.ClusterName, "pool2"),
	}).Return(nil).Once()

	s.env.OnActivity(DeleteControlPlaneActivityName, mock.Anything, DeleteControlPlaneActivityInput{
		EKSActivityInput: eksActivityInput,
	}).Return(nil).Once()

	s.env.OnActivity(DeleteSshKeyActivityName, mock.Anything, DeleteSshKeyActivityInput{
		EKSActivityInput: eksActivityInput,
		SSHKeyName:       generateSSHKeyNameForCluster(eksActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.OnActivity(GetOrphanNICsActivityName, mock.Anything, GetOrphanNICsActivityInput{
		EKSActivityInput: eksActivityInput,
		VpcID:            "test-vpc-id",
		SecurityGroupIDs: []string{"test-node-sg-id", "test-control-plane-sg-id"},
	}).Return(&GetOrphanNICsActivityOutput{
		NicList: []string{
			"nic1",
		},
	}, nil).Once()

	s.env.OnActivity(DeleteOrphanNICActivityName, mock.Anything, DeleteOrphanNICActivityInput{
		EKSActivityInput: eksActivityInput,
		NicID:            "nic1",
	}).Return(nil).Once()

	s.env.OnActivity(GetSubnetStacksActivityName, mock.Anything, GetSubnetStacksActivityInput{
		EKSActivityInput: eksActivityInput,
	}).Return(&GetSubnetStacksActivityOutput{
		StackNames: []string{
			"subnetStackName",
		},
	}, nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        "subnetStackName",
	}).Return(nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        generateStackNameForCluster(eksActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        generateStackNameForIam(eksActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.OnActivity(DeleteClusterUserAccessKeyActivityName, mock.Anything, DeleteClusterUserAccessKeyActivityInput{
		EKSActivityInput: eksActivityInput,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(DeleteInfraWorkflowName, workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
