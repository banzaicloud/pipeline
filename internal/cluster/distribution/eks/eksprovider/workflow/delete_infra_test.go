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

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

type DeleteClusterInfraWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestDeleteClusterInfraWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteClusterInfraWorkflowTestSuite))
}

func (s *DeleteClusterInfraWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()

	s.env.RegisterWorkflowWithOptions(DeleteInfrastructureWorkflow, workflow.RegisterOptions{Name: DeleteInfraWorkflowName})

	deleteNodePoolWorkflow := NewDeleteNodePoolWorkflow()
	s.env.RegisterWorkflowWithOptions(deleteNodePoolWorkflow.Execute, workflow.RegisterOptions{Name: DeleteNodePoolWorkflowName})

	getVpcConfigActivity := NewGetVpcConfigActivity(nil)
	s.env.RegisterActivityWithOptions(getVpcConfigActivity.Execute, activity.RegisterOptions{Name: GetVpcConfigActivityName})

	getOwnedELBsActivity := NewGetOwnedELBsActivity(nil)
	s.env.RegisterActivityWithOptions(getOwnedELBsActivity.Execute, activity.RegisterOptions{Name: GetOwnedELBsActivityName})

	waitELBsDeletionActivity := NewWaitELBsDeletionActivity(nil)
	s.env.RegisterActivityWithOptions(waitELBsDeletionActivity.Execute, activity.RegisterOptions{Name: WaitELBsDeletionActivityName})

	deleteStackActivity := NewDeleteStackActivity(nil)
	s.env.RegisterActivityWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: DeleteStackActivityName})

	deleteControlPlaneActivity := NewDeleteControlPlaneActivity(nil)
	s.env.RegisterActivityWithOptions(deleteControlPlaneActivity.Execute, activity.RegisterOptions{Name: DeleteControlPlaneActivityName})

	deleteSshKeyActivity := NewDeleteSshKeyActivity(nil)
	s.env.RegisterActivityWithOptions(deleteSshKeyActivity.Execute, activity.RegisterOptions{Name: DeleteSshKeyActivityName})

	getOrphanNicsActivity := NewGetOrphanNICsActivity(nil)
	s.env.RegisterActivityWithOptions(getOrphanNicsActivity.Execute, activity.RegisterOptions{Name: GetOrphanNICsActivityName})

	deleteOrphanNicActivity := NewDeleteOrphanNICActivity(nil)
	s.env.RegisterActivityWithOptions(deleteOrphanNicActivity.Execute, activity.RegisterOptions{Name: DeleteOrphanNICActivityName})

	getSubnetStacksActivity := NewGetSubnetStacksActivity(nil)
	s.env.RegisterActivityWithOptions(getSubnetStacksActivity.Execute, activity.RegisterOptions{Name: GetSubnetStacksActivityName})
}

func (s *DeleteClusterInfraWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *DeleteClusterInfraWorkflowTestSuite) Test_Successful_Delete_Infra() {
	workflowInput := DeleteInfrastructureWorkflowInput{
		Region:           "us-west-1",
		OrganizationID:   1,
		SecretID:         "my-secret-id",
		ClusterID:        1,
		ClusterName:      "test-cluster-name",
		NodePoolNames:    []string{"pool1", "pool2"},
		GeneratedSSHUsed: true,
	}

	AWSCommonActivityInput := awsworkflow.AWSCommonActivityInput{
		OrganizationID:            workflowInput.OrganizationID,
		SecretID:                  workflowInput.SecretID,
		Region:                    workflowInput.Region,
		ClusterName:               workflowInput.ClusterName,
		AWSClientRequestTokenBase: "default-test-workflow-id",
	}

	s.env.OnActivity(GetVpcConfigActivityName, mock.Anything, GetVpcConfigActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		StackName:              "pipeline-eks-test-cluster-name",
	}).Return(&GetVpcConfigActivityOutput{
		VpcID:               "test-vpc-id",
		SecurityGroupID:     "test-control-plane-sg-id",
		NodeSecurityGroupID: "test-node-sg-id",
	}, nil)

	s.env.OnActivity(GetOwnedELBsActivityName, mock.Anything, GetOwnedELBsActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		VpcID:                  "test-vpc-id",
	}).Return(&GetOwnedELBsActivityOutput{
		LoadBalancerNames: []string{"test-lb-1", "test-lb-2"},
	}, nil)

	s.env.OnActivity(WaitELBsDeletionActivityName, mock.Anything, WaitELBsDeletionActivityActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		LoadBalancerNames:      []string{"test-lb-1", "test-lb-2"},
	}).Return(nil)

	for _, nodePoolName := range workflowInput.NodePoolNames {
		s.env.OnWorkflow(DeleteNodePoolWorkflowName, mock.Anything, DeleteNodePoolWorkflowInput{
			ClusterID:                 workflowInput.ClusterID,
			ClusterName:               workflowInput.ClusterName,
			NodePoolName:              nodePoolName,
			OrganizationID:            workflowInput.OrganizationID,
			Region:                    workflowInput.Region,
			SecretID:                  workflowInput.SecretID,
			ShouldUpdateClusterStatus: false,
		}).Return(nil).Once()
	}

	s.env.OnActivity(DeleteControlPlaneActivityName, mock.Anything, DeleteControlPlaneActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
	}).Return(nil).Once()

	s.env.OnActivity(DeleteSshKeyActivityName, mock.Anything, DeleteSshKeyActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		SSHKeyName:             GenerateSSHKeyNameForCluster(AWSCommonActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.OnActivity(GetOrphanNICsActivityName, mock.Anything, GetOrphanNICsActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		VpcID:                  "test-vpc-id",
		SecurityGroupIDs:       []string{"test-node-sg-id", "test-control-plane-sg-id"},
	}).Return(&GetOrphanNICsActivityOutput{
		NicList: []string{
			"nic1",
		},
	}, nil).Once()

	s.env.OnActivity(DeleteOrphanNICActivityName, mock.Anything, DeleteOrphanNICActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		NicID:                  "nic1",
	}).Return(nil).Once()

	s.env.OnActivity(GetSubnetStacksActivityName, mock.Anything, GetSubnetStacksActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
	}).Return(&GetSubnetStacksActivityOutput{
		StackNames: []string{
			"subnetStackName",
		},
	}, nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		StackName:              "subnetStackName",
	}).Return(nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		StackName:              GenerateStackNameForCluster(AWSCommonActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.OnActivity(DeleteStackActivityName, mock.Anything, DeleteStackActivityInput{
		AWSCommonActivityInput: AWSCommonActivityInput,
		StackName:              generateStackNameForIam(AWSCommonActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(DeleteInfraWorkflowName, workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
