// Copyright © 2021 Banzai Cloud
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

package cadence

import (
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/mocks"
	"go.uber.org/cadence/workflow"
)

func TestNewCronConfiguration(t *testing.T) {
	type inputType struct {
		CadenceClient                client.Client
		CronInstanceType             CronInstanceType
		CronSchedule                 string
		ExecutionStartToCloseTimeout time.Duration
		TaskListName                 string
		Workflow                     string
		WorkflowArguments            []interface{}
	}

	type caseType struct {
		caseDescription           string
		expectedCronConfiguration CronConfiguration
		input                     inputType
	}

	testCases := []caseType{
		{
			caseDescription: "non-zero values -> non-zero configuration",
			expectedCronConfiguration: CronConfiguration{
				CadenceClient:                &mocks.Client{},
				CronInstanceType:             CronInstanceTypeDomain,
				CronSchedule:                 "0/1 * * * *",
				ExecutionStartToCloseTimeout: time.Second,
				TaskListName:                 "task-list-name",
				Workflow:                     "workflow",
				WorkflowArguments: []interface{}{
					false,
					1,
					"2",
				},
			},
			input: inputType{
				CadenceClient:                &mocks.Client{},
				CronInstanceType:             CronInstanceTypeDomain,
				CronSchedule:                 "0/1 * * * *",
				ExecutionStartToCloseTimeout: time.Second,
				TaskListName:                 "task-list-name",
				Workflow:                     "workflow",
				WorkflowArguments: []interface{}{
					false,
					1,
					"2",
				},
			},
		},
		{
			caseDescription:           "zero values -> zero configuration",
			expectedCronConfiguration: CronConfiguration{},
			input:                     inputType{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualCronConfiguration := NewCronConfiguration(
				testCase.input.CadenceClient,
				testCase.input.CronInstanceType,
				testCase.input.CronSchedule,
				testCase.input.ExecutionStartToCloseTimeout,
				testCase.input.TaskListName,
				testCase.input.Workflow,
				testCase.input.WorkflowArguments...,
			)

			require.Equal(t, testCase.expectedCronConfiguration, actualCronConfiguration)
		})
	}
}

func TestCronConfigurationCronWorkflowID(t *testing.T) {
	type caseType struct {
		caseDescription        string
		expectedCronWorkflowID string
		inputCronConfiguration CronConfiguration
	}

	testCases := []caseType{
		{
			caseDescription:        "domain cron instance type -> task list cron workflow id",
			expectedCronWorkflowID: "task-list-name-cron-workflow",
			inputCronConfiguration: NewCronConfiguration(
				&mocks.Client{},
				CronInstanceTypeDomain,
				"0/1 * * * *",
				time.Second,
				"task-list-name",
				"workflow",
			),
		},
		{
			caseDescription:        "unknown cron instance type -> instance type cron workflow id",
			expectedCronWorkflowID: "unknown-instance-type-cron-workflow",
			inputCronConfiguration: NewCronConfiguration(
				&mocks.Client{},
				CronInstanceType("unknown-instance-type"),
				"0/1 * * * *",
				time.Second,
				"task-list-name",
				"workflow",
			),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualCronWorkflowID := testCase.inputCronConfiguration.CronWorkflowID()

			require.Equal(t, testCase.expectedCronWorkflowID, actualCronWorkflowID)
		})
	}
}

func TestCronConfigurationStartCronWorkflow(t *testing.T) {
	type inputType struct {
		cronConfig CronConfiguration
		ctx        context.Context
	}

	type mocksType struct {
		workflowExecutionDescription      *shared.DescribeWorkflowExecutionResponse
		workflowExecutionDescriptionError error
		workflowTerminationError          error
		workflowStartError                error
	}

	type caseType struct {
		caseDescription string
		expectedError   error
		input           inputType
		mocks           mocksType
	}

	testCases := []caseType{
		{
			caseDescription: "state error -> error",
			expectedError: errors.New(
				"querying workflow state failed" +
					": failed to query cron workflow" +
					": test error",
			),
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription:      nil,
				workflowExecutionDescriptionError: errors.New("test error"),
			},
		},
		{
			caseDescription: "scheduled state -> nothing to do",
			expectedError:   nil,
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("\"0/1 * * * *\""),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
			},
		},
		{
			caseDescription: "scheduled outdated with termination error -> error",
			expectedError:   errors.New("terminating cron workflow failed: test workflow termination error"),
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("\"* 0/2 * * *\""),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
				workflowTerminationError:          errors.New("test workflow termination error"),
			},
		},
		{
			caseDescription: "scheduled outdated with start error -> error",
			expectedError:   errors.New("starting cron workflow failed: test workflow start error"),
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("\"* 0/2 * * *\""),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
				workflowStartError:                errors.New("test workflow start error"),
			},
		},
		{
			caseDescription: "scheduled outdated no error -> success",
			expectedError:   nil,
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("\"* 0/2 * * *\""),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
			},
		},
		{
			caseDescription: "not existing start error -> error",
			expectedError:   errors.New("starting cron workflow failed: test workflow start error"),
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription:      nil,
				workflowExecutionDescriptionError: &shared.EntityNotExistsError{},
				workflowStartError:                errors.New("test workflow start error"),
			},
		},
		{
			caseDescription: "not existing no error -> success",
			expectedError:   nil,
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription:      nil,
				workflowExecutionDescriptionError: &shared.EntityNotExistsError{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			cronWorkflowID := testCase.input.cronConfig.CronWorkflowID()
			cadenceClientMock := testCase.input.cronConfig.CadenceClient.(*mocks.Client)
			cadenceClientMock.On("DescribeWorkflowExecution", testCase.input.ctx, cronWorkflowID, "").
				Return(testCase.mocks.workflowExecutionDescription, testCase.mocks.workflowExecutionDescriptionError).
				Once()

			activeCronSchedule := ""
			if testCase.mocks.workflowExecutionDescriptionError == nil {
				executionInfo := testCase.mocks.workflowExecutionDescription.WorkflowExecutionInfo
				err := client.NewValue(executionInfo.Memo.Fields["CronSchedule"]).Get(&activeCronSchedule)
				require.NoError(t, err)
			}

			if (testCase.mocks.workflowExecutionDescriptionError == nil ||
				errors.As(testCase.mocks.workflowExecutionDescriptionError, new(*shared.EntityNotExistsError))) &&
				activeCronSchedule != testCase.input.cronConfig.CronSchedule {
				if activeCronSchedule != "" {
					cadenceClientMock.On(
						"TerminateWorkflow",
						testCase.input.ctx,
						cronWorkflowID,
						"",
						"cron workflow schedule requires an update",
						([]byte)(nil),
					).Return(testCase.mocks.workflowTerminationError).Once()
				}

				if testCase.mocks.workflowTerminationError == nil {
					workflowOptions := client.StartWorkflowOptions{
						ID:                           cronWorkflowID,
						TaskList:                     testCase.input.cronConfig.TaskListName,
						ExecutionStartToCloseTimeout: testCase.input.cronConfig.ExecutionStartToCloseTimeout,
						CronSchedule:                 testCase.input.cronConfig.CronSchedule,
						Memo: map[string]interface{}{ // Note: CronSchedule is not directly retrievable (version 0.13.4-0.15.0).
							"CronSchedule": testCase.input.cronConfig.CronSchedule,
						},
					}
					cadenceClientMock.On(
						"StartWorkflow",
						append(
							[]interface{}{
								testCase.input.ctx,
								workflowOptions,
								testCase.input.cronConfig.Workflow,
							},
							testCase.input.cronConfig.WorkflowArguments...,
						)...,
					).Return(&workflow.Execution{}, testCase.mocks.workflowStartError).Once()
				}
			}

			actualError := testCase.input.cronConfig.StartCronWorkflow(testCase.input.ctx)

			cadenceClientMock.AssertExpectations(t)

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestCronConfigurationWorkflowState(t *testing.T) {
	type inputType struct {
		cronConfig CronConfiguration
		ctx        context.Context
	}

	type mocksType struct {
		workflowExecutionDescription      *shared.DescribeWorkflowExecutionResponse
		workflowExecutionDescriptionError error
	}

	type outputType struct {
		expectedWorkflowState CronWorkflowState
		expectedError         error
	}

	type caseType struct {
		caseDescription string
		input           inputType
		mocks           mocksType
		output          outputType
	}

	testCases := []caseType{
		{
			caseDescription: "not existing -> not scheduled state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription:      nil,
				workflowExecutionDescriptionError: &shared.EntityNotExistsError{},
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateNotScheduled,
				expectedError:         nil,
			},
		},
		{
			caseDescription: "description error -> unknown state with error",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription:      nil,
				workflowExecutionDescriptionError: errors.New("test error"),
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateUnknown,
				expectedError:         errors.New("failed to query cron workflow: test error"),
			},
		},
		{
			caseDescription: "nil workflow execution info -> unknown state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: nil,
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateUnknown,
				expectedError:         errors.New("cron workflow execution information not found"),
			},
		},
		{
			caseDescription: "canceled workflow -> not scheduled state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						CloseStatus: func() *shared.WorkflowExecutionCloseStatus {
							closeStatus := shared.WorkflowExecutionCloseStatusCanceled

							return &closeStatus
						}(),
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateNotScheduled,
				expectedError:         nil,
			},
		},
		{
			caseDescription: "terminated workflow -> not scheduled state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						CloseStatus: func() *shared.WorkflowExecutionCloseStatus {
							closeStatus := shared.WorkflowExecutionCloseStatusTerminated

							return &closeStatus
						}(),
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateNotScheduled,
				expectedError:         nil,
			},
		},
		{
			caseDescription: "no memo -> scheduled outdated state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: nil,
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateScheduledOutdated,
				expectedError:         nil,
			},
		},
		{
			caseDescription: "no cron schedule -> scheduled outdated state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateScheduledOutdated,
				expectedError:         nil,
			},
		},
		{
			caseDescription: "invalid cron schedule -> unknown state with error",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("not-a-valid-cron-schedule"),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateUnknown,
				expectedError: errors.New(
					"retrieving cron schedule failed" +
						": unable to decode argument: 0, *string, with json error" +
						": invalid character 'o' in literal null (expecting 'u')"),
			},
		},
		{
			caseDescription: "mismatching cron schedule -> scheduled outdated state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("\"* 0/2 * * *\""),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateScheduledOutdated,
				expectedError:         nil,
			},
		},
		{
			caseDescription: "matching cron schedule -> scheduled state",
			input: inputType{
				cronConfig: NewCronConfiguration(
					&mocks.Client{},
					CronInstanceTypeDomain,
					"0/1 * * * *",
					time.Second,
					"task-list-name",
					"workflow",
				),
				ctx: context.Background(),
			},
			mocks: mocksType{
				workflowExecutionDescription: &shared.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
						Memo: &shared.Memo{
							Fields: map[string][]byte{
								"CronSchedule": []byte("\"0/1 * * * *\""),
							},
						},
					},
				},
				workflowExecutionDescriptionError: nil,
			},
			output: outputType{
				expectedWorkflowState: CronWorkflowStateScheduled,
				expectedError:         nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			cronWorkflowID := testCase.input.cronConfig.CronWorkflowID()
			cadenceClientMock := testCase.input.cronConfig.CadenceClient.(*mocks.Client)
			cadenceClientMock.On("DescribeWorkflowExecution", testCase.input.ctx, cronWorkflowID, "").
				Return(testCase.mocks.workflowExecutionDescription, testCase.mocks.workflowExecutionDescriptionError).
				Once()

			actualWorkflowState, actualError := testCase.input.cronConfig.WorkflowState(testCase.input.ctx)

			cadenceClientMock.AssertExpectations(t)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedWorkflowState, actualWorkflowState)
		})
	}
}
