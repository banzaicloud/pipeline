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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/banzaicloud/cadence-aws-sdk/clients/ec2stub"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow/awsmocks"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

type DeleteClusterInfraWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment

	ec2client *awsmocks.Mockec2clientstub
}

func TestDeleteClusterInfraWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteClusterInfraWorkflowTestSuite))
}

func (s *DeleteClusterInfraWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()

	s.ec2client = new(awsmocks.Mockec2clientstub)

	NewDeleteInfrastructureWorkflow(s.ec2client).Register(s.env)

	deleteNodePoolWorkflow := NewDeleteNodePoolWorkflow()
	s.env.RegisterWorkflowWithOptions(deleteNodePoolWorkflow.Execute, workflow.RegisterOptions{Name: DeleteNodePoolWorkflowName})

	getVpcConfigActivity := NewGetVpcConfigActivity(nil)
	s.env.RegisterActivityWithOptions(getVpcConfigActivity.Execute, activity.RegisterOptions{Name: GetVpcConfigActivityName})

	getOwnedELBsActivity := NewGetOwnedELBsActivity(nil)
	s.env.RegisterActivityWithOptions(getOwnedELBsActivity.Execute, activity.RegisterOptions{Name: GetOwnedELBsActivityName})

	waitELBsDeletionActivity := NewWaitELBsDeletionActivity(nil)
	s.env.RegisterActivityWithOptions(waitELBsDeletionActivity.Execute, activity.RegisterOptions{Name: WaitELBsDeletionActivityName})

	deleteStackActivity := awsworkflow.NewDeleteStackActivity(nil)
	s.env.RegisterActivityWithOptions(deleteStackActivity.Execute, activity.RegisterOptions{Name: awsworkflow.DeleteStackActivityName})

	deleteControlPlaneActivity := NewDeleteControlPlaneActivity(nil)
	s.env.RegisterActivityWithOptions(deleteControlPlaneActivity.Execute, activity.RegisterOptions{Name: DeleteControlPlaneActivityName})

	getOrphanNicsActivity := NewGetOrphanNICsActivity(nil)
	s.env.RegisterActivityWithOptions(getOrphanNicsActivity.Execute, activity.RegisterOptions{Name: GetOrphanNICsActivityName})

	deleteOrphanNicActivity := NewDeleteOrphanNICActivity(nil)
	s.env.RegisterActivityWithOptions(deleteOrphanNicActivity.Execute, activity.RegisterOptions{Name: DeleteOrphanNICActivityName})

	getSubnetStacksActivity := NewGetSubnetStacksActivity(nil)
	s.env.RegisterActivityWithOptions(getSubnetStacksActivity.Execute, activity.RegisterOptions{Name: GetSubnetStacksActivityName})
}

func (s *DeleteClusterInfraWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
	s.ec2client.AssertExpectations(s.T())
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

	eksActivityInput := EKSActivityInput{
		OrganizationID: workflowInput.OrganizationID,
		SecretID:       workflowInput.SecretID,
		Region:         workflowInput.Region,
		ClusterName:    workflowInput.ClusterName,
	}

	awsCommonActivityInput := awsworkflow.AWSCommonActivityInput{
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
		EKSActivityInput: eksActivityInput,
	}).Return(nil).Once()

	s.ec2client.On("DeleteKeyPairAsync", mock.Anything, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(GenerateSSHKeyNameForCluster(eksActivityInput.ClusterName)),
	}).Return(&ec2stub.DeleteKeyPairFuture{
		Future: future{},
	})

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

	s.env.OnActivity(awsworkflow.DeleteStackActivityName, mock.Anything, awsworkflow.DeleteStackActivityInput{
		AWSCommonActivityInput: awsCommonActivityInput,
		StackName:              "subnetStackName",
	}).Return(nil).Once()

	s.env.OnActivity(awsworkflow.DeleteStackActivityName, mock.Anything, awsworkflow.DeleteStackActivityInput{
		AWSCommonActivityInput: awsCommonActivityInput,
		StackName:              GenerateStackNameForCluster(eksActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.OnActivity(awsworkflow.DeleteStackActivityName, mock.Anything, awsworkflow.DeleteStackActivityInput{
		AWSCommonActivityInput: awsCommonActivityInput,
		StackName:              generateStackNameForIam(awsCommonActivityInput.ClusterName),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(DeleteInfraWorkflowName, workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

type future struct{}

func (f future) Get(ctx workflow.Context, valuePtr interface{}) error {
	return nil
}

func (f future) IsReady() bool {
	return true
}
