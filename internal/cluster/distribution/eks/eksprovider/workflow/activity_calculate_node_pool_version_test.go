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

	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

func TestCalculateNodePoolVersion(t *testing.T) {
	type inputType struct {
		input CalculateNodePoolVersionActivityInput
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				input: CalculateNodePoolVersionActivityInput{
					Image: "ami-xxxxxxxxxxxxx",
					VolumeEncryption: &eks.NodePoolVolumeEncryption{
						Enabled:          true,
						EncryptionKeyARN: "arn:aws:kms:region:account:key/id",
					},
					VolumeSize: 50,
					CustomSecurityGroups: []string{
						"sg-1",
						"sg-2",
					},
				},
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				input: CalculateNodePoolVersionActivityInput{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input CalculateNodePoolVersionActivityInput) (*CalculateNodePoolVersionActivityOutput, error) {
					return &CalculateNodePoolVersionActivityOutput{}, testCase.expectedError
				},
				activity.RegisterOptions{Name: CalculateNodePoolVersionActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				_, actualError = calculateNodePoolVersion(
					ctx,
					testCase.input.input.Image,
					testCase.input.input.VolumeEncryption,
					testCase.input.input.VolumeSize,
					testCase.input.input.CustomSecurityGroups,
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
