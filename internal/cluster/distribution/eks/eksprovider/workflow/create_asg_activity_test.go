// Copyright © 2020 Banzai Cloud
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
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

func TestNewCreateASGActivity(t *testing.T) {
	type inputType struct {
		awsSessionFactory           awsworkflow.AWSFactory
		cloudFormationTemplate      string
		defaultNodeVolumeEncryption *eks.NodePoolVolumeEncryption
		nodePoolStore               eks.NodePoolStore
	}

	testCases := []struct {
		caseDescription  string
		expectedActivity *CreateAsgActivity
		input            inputType
	}{
		{
			caseDescription:  "nil values -> success",
			expectedActivity: &CreateAsgActivity{},
			input:            inputType{},
		},
		{
			caseDescription: "not nil values -> success",
			expectedActivity: &CreateAsgActivity{
				awsSessionFactory:      &awsworkflow.MockAWSFactory{},
				cloudFormationTemplate: "cloudformation-template",
				defaultNodeVolumeEncryption: &eks.NodePoolVolumeEncryption{
					Enabled:          true,
					EncryptionKeyARN: "encryption-key",
				},
				nodePoolStore: &eks.MockNodePoolStore{},
			},
			input: inputType{
				awsSessionFactory:      &awsworkflow.MockAWSFactory{},
				cloudFormationTemplate: "cloudformation-template",
				defaultNodeVolumeEncryption: &eks.NodePoolVolumeEncryption{
					Enabled:          true,
					EncryptionKeyARN: "encryption-key",
				},
				nodePoolStore: &eks.MockNodePoolStore{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewCreateAsgActivity(
				testCase.input.awsSessionFactory,
				testCase.input.cloudFormationTemplate,
				testCase.input.defaultNodeVolumeEncryption,
				testCase.input.nodePoolStore,
			)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestCreateASG(t *testing.T) {
	type inputType struct {
		eksActivityInput   EKSActivityInput
		eksCluster         eksmodel.EKSClusterModel
		vpcConfig          GetVpcConfigActivityOutput
		nodePool           eks.NewNodePool
		nodePoolSubnetIDs  []string
		selectedVolumeSize int
		nodePoolVersion    string
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				eksCluster: eksmodel.EKSClusterModel{
					Subnets: []*eksmodel.EKSSubnetModel{
						{
							SubnetId: aws.String("subnet-id"),
						},
						{
							SubnetId: aws.String("subnet-id-2"),
						},
					},
				},
				vpcConfig: GetVpcConfigActivityOutput{},
				nodePool: eks.NewNodePool{
					SubnetID: "subnet-id",
				},
				nodePoolSubnetIDs: []string{
					"subnet-id-2",
				},
				selectedVolumeSize: 1,
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				eksCluster: eksmodel.EKSClusterModel{
					Subnets: []*eksmodel.EKSSubnetModel{
						{
							SubnetId: aws.String("subnet-id"),
						},
						{
							SubnetId: aws.String("subnet-id-2"),
						},
					},
				},
				vpcConfig: GetVpcConfigActivityOutput{},
				nodePool: eks.NewNodePool{
					SubnetID: "subnet-id",
				},
				nodePoolSubnetIDs: []string{
					"subnet-id-2",
				},
				selectedVolumeSize: 1,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input CreateAsgActivityInput) (*CreateAsgActivityOutput, error) {
					return &CreateAsgActivityOutput{}, testCase.expectedError
				},
				activity.RegisterOptions{Name: CreateAsgActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualError = createASG(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.eksCluster,
					testCase.input.vpcConfig,
					testCase.input.nodePool,
					testCase.input.nodePoolSubnetIDs,
					testCase.input.selectedVolumeSize,
					testCase.input.nodePoolVersion,
				)

				return nil
			})

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestCreateASGAsync(t *testing.T) {
	type inputType struct {
		eksActivityInput   EKSActivityInput
		eksCluster         eksmodel.EKSClusterModel
		vpcConfig          GetVpcConfigActivityOutput
		nodePool           eks.NewNodePool
		nodePoolSubnetIDs  []string
		selectedVolumeSize int
		nodePoolVersion    string
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "activity error",
			expectedError:   errors.New("test error"),
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				eksCluster: eksmodel.EKSClusterModel{
					Subnets: []*eksmodel.EKSSubnetModel{
						{
							SubnetId: aws.String("subnet-id"),
						},
						{
							SubnetId: aws.String("subnet-id-2"),
						},
					},
				},
				vpcConfig: GetVpcConfigActivityOutput{},
				nodePool: eks.NewNodePool{
					SubnetID: "subnet-id",
				},
				nodePoolSubnetIDs: []string{
					"subnet-id-2",
				},
				selectedVolumeSize: 1,
			},
		},
		{
			caseDescription: "subnet error",
			expectedError:   errors.New("node pool subnets could not be determined: some subnet IDs could not be found among the subnets"),
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				eksCluster: eksmodel.EKSClusterModel{
					NodeInstanceRoleId: "/test/pathed/node-instance-role-id",
					SSHGenerated:       true,
					Subnets: []*eksmodel.EKSSubnetModel{
						{
							SubnetId: aws.String("subnet-id"),
						},
					},
				},
				vpcConfig: GetVpcConfigActivityOutput{},
				nodePool: eks.NewNodePool{
					SubnetID: "subnet-id",
				},
				nodePoolSubnetIDs: []string{
					"subnet-id-2",
				},
				selectedVolumeSize: 1,
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				eksCluster: eksmodel.EKSClusterModel{
					SSHGenerated: true,
					Subnets: []*eksmodel.EKSSubnetModel{
						{
							SubnetId: aws.String("subnet-id"),
						},
						{
							SubnetId: aws.String("subnet-id-2"),
						},
					},
				},
				vpcConfig: GetVpcConfigActivityOutput{},
				nodePool: eks.NewNodePool{
					Autoscaling: eks.Autoscaling{
						Enabled: true,
					},
					SubnetID: "subnet-id",
				},
				nodePoolSubnetIDs: []string{
					"subnet-id-2",
				},
				selectedVolumeSize: 1,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input CreateAsgActivityInput) (*CreateAsgActivityOutput, error) {
					return &CreateAsgActivityOutput{}, testCase.expectedError
				},
				activity.RegisterOptions{Name: CreateAsgActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := createASGAsync(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.eksCluster,
					testCase.input.vpcConfig,
					testCase.input.nodePool,
					testCase.input.nodePoolSubnetIDs,
					testCase.input.selectedVolumeSize,
					testCase.input.nodePoolVersion,
				)
				actualError = actualFuture.Get(ctx, nil)

				return nil
			})

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}
