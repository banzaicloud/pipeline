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
	"strings"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

func TestNewGetAMISizeActivity(t *testing.T) {
	type inputType struct {
		awsFactory awsworkflow.AWSFactory
		ec2Factory EC2APIFactory
	}

	testCases := []struct {
		caseDescription  string
		expectedActivity *GetAMISizeActivity
		input            inputType
	}{
		{
			caseDescription:  "nil values -> success",
			expectedActivity: &GetAMISizeActivity{},
			input:            inputType{},
		},
		{
			caseDescription: "not nil values -> success",
			expectedActivity: &GetAMISizeActivity{
				awsFactory: &awsworkflow.MockAWSFactory{},
				ec2Factory: &MockEC2APIFactory{},
			},
			input: inputType{
				awsFactory: &awsworkflow.MockAWSFactory{},
				ec2Factory: &MockEC2APIFactory{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewGetAMISizeActivity(testCase.input.awsFactory, testCase.input.ec2Factory)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestGetAMISizeActivityExecute(t *testing.T) {
	type inputType struct {
		activity *GetAMISizeActivity
		input    GetAMISizeActivityInput
	}

	type outputType struct {
		expectedError  error
		expectedOutput *GetAMISizeActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "aws factory error -> error",
			input: inputType{
				activity: NewGetAMISizeActivity(&awsworkflow.MockAWSFactory{}, &MockEC2APIFactory{}),
				input: GetAMISizeActivityInput{
					EKSActivityInput: EKSActivityInput{},
					ImageID:          "image-id",
				},
			},
			output: outputType{
				expectedError:  errors.New("aws factory error"),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "describe images error -> error",
			input: inputType{
				activity: NewGetAMISizeActivity(&awsworkflow.MockAWSFactory{}, &MockEC2APIFactory{}),
				input: GetAMISizeActivityInput{
					EKSActivityInput: EKSActivityInput{},
					ImageID:          "image-id",
				},
			},
			output: outputType{
				expectedError:  errors.New("describing AMI failed: "),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "image not found error -> error",
			input: inputType{
				activity: NewGetAMISizeActivity(&awsworkflow.MockAWSFactory{}, &MockEC2APIFactory{}),
				input: GetAMISizeActivityInput{
					EKSActivityInput: EKSActivityInput{},
					ImageID:          "image-id",
				},
			},
			output: outputType{
				expectedError:  errors.New("describing AMI found no record"),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "block device mapping not found error -> error",
			input: inputType{
				activity: NewGetAMISizeActivity(&awsworkflow.MockAWSFactory{}, &MockEC2APIFactory{}),
				input: GetAMISizeActivityInput{
					EKSActivityInput: EKSActivityInput{},
					ImageID:          "image-id",
				},
			},
			output: outputType{
				expectedError:  errors.New("describing AMI found no block device mappings"),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "aws factory error -> error",
			input: inputType{
				activity: NewGetAMISizeActivity(&awsworkflow.MockAWSFactory{}, &MockEC2APIFactory{}),
				input: GetAMISizeActivityInput{
					EKSActivityInput: EKSActivityInput{},
					ImageID:          "image-id",
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &GetAMISizeActivityOutput{
					AMISize: 1,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			mockAWSFactory := testCase.input.activity.awsFactory.(*awsworkflow.MockAWSFactory)
			awsFactoryNewMock := mockAWSFactory.On(
				"New",
				testCase.input.input.EKSActivityInput.OrganizationID,
				testCase.input.input.EKSActivityInput.SecretID,
				testCase.input.input.EKSActivityInput.Region,
			)
			if testCase.output.expectedError != nil &&
				testCase.output.expectedError.Error() == "aws factory error" {
				awsFactoryNewMock.Return(nil, testCase.output.expectedError).Once()
			} else {
				awsFactoryNewMock.Return((*session.Session)(nil), nil)
			}

			mockEC2Client := &Mockec2API{}
			mockEC2Factory := testCase.input.activity.ec2Factory.(*MockEC2APIFactory)
			mockEC2Factory.On("New", (*session.Session)(nil)).Return(mockEC2Client)

			ec2ClientDescribeImagesMock := mockEC2Client.On(
				"DescribeImages",
				&ec2.DescribeImagesInput{
					ImageIds: []*string{
						aws.String(testCase.input.input.ImageID),
					},
				},
			)
			if testCase.output.expectedError != nil &&
				strings.HasPrefix(testCase.output.expectedError.Error(), "describing AMI failed") {
				ec2ClientDescribeImagesMock.Return(nil, errors.New(""))
			} else if testCase.output.expectedError != nil &&
				strings.HasPrefix(testCase.output.expectedError.Error(), "describing AMI found no record") {
				ec2ClientDescribeImagesMock.Return(&ec2.DescribeImagesOutput{}, nil)
			} else if testCase.output.expectedError != nil &&
				testCase.output.expectedError.Error() == "describing AMI found no block device mappings" {
				ec2ClientDescribeImagesMock.Return(&ec2.DescribeImagesOutput{
					Images: []*ec2.Image{
						{},
					},
				}, nil)
			} else if testCase.output.expectedOutput != nil {
				ec2ClientDescribeImagesMock.Return(&ec2.DescribeImagesOutput{
					Images: []*ec2.Image{
						{
							BlockDeviceMappings: []*ec2.BlockDeviceMapping{
								{
									Ebs: &ec2.EbsBlockDevice{
										VolumeSize: aws.Int64(int64(testCase.output.expectedOutput.AMISize)),
									},
								},
							},
						},
					},
				}, nil)
			}

			actualOutput, actualError := testCase.input.activity.Execute(
				context.Background(),
				testCase.input.input,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}

func TestGetAMISize(t *testing.T) {
	type inputType struct {
		eksActivityInput EKSActivityInput
		imageID          string
	}

	type outputType struct {
		expectedError   error
		expectedAMISize int
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				imageID:          "image-id",
			},
			output: outputType{
				expectedError:   nil,
				expectedAMISize: 1,
			},
		},
		{
			caseDescription: "error",
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				imageID:          "image-id",
			},
			output: outputType{
				expectedError:   errors.New("test error"),
				expectedAMISize: 0,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetAMISizeActivityInput) (*GetAMISizeActivityOutput, error) {
					return &GetAMISizeActivityOutput{
						AMISize: testCase.output.expectedAMISize,
					}, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetAMISizeActivityName},
			)

			var actualError error
			var actualAMISize int
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualAMISize, actualError = getAMISize(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.imageID,
				)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedAMISize, actualAMISize)
		})
	}
}

func TestGetAMISizeAsync(t *testing.T) {
	type inputType struct {
		eksActivityInput EKSActivityInput
		imageID          string
	}

	type outputType struct {
		expectedError  error
		expectedOutput *GetAMISizeActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				imageID:          "image-id",
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &GetAMISizeActivityOutput{
					AMISize: 1,
				},
			},
		},
		{
			caseDescription: "error",
			input: inputType{
				eksActivityInput: EKSActivityInput{},
				imageID:          "image-id",
			},
			output: outputType{
				expectedError:  errors.New("test error"),
				expectedOutput: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetAMISizeActivityInput) (*GetAMISizeActivityOutput, error) {
					return testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetAMISizeActivityName},
			)

			var actualError error
			var actualOutput *GetAMISizeActivityOutput
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := getAMISizeAsync(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.imageID,
				)
				actualError = actualFuture.Get(ctx, &actualOutput)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}
