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

package cadence

import (
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
)

func TestNewReadyFuture(t *testing.T) {
	type inputType struct {
		err   error
		value interface{}
	}

	type outputType struct {
		expectedError error
		expectedValue interface{}
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "nil value, nil error -> success",
			input: inputType{
				err:   nil,
				value: nil,
			},
			output: outputType{
				expectedError: nil,
				expectedValue: nil,
			},
		},
		{
			caseDescription: "nil value, not nil error -> success",
			input: inputType{
				err:   errors.New("test error"),
				value: nil,
			},
			output: outputType{
				expectedError: errors.New("test error"),
				expectedValue: nil,
			},
		},
		{
			caseDescription: "not nil value, nil error -> success",
			input: inputType{
				err:   nil,
				value: "value",
			},
			output: outputType{
				expectedError: nil,
				expectedValue: "value",
			},
		},
		{
			caseDescription: "not nil value, not nil error -> success",
			input: inputType{
				err:   errors.New("test error"),
				value: "value",
			},
			output: outputType{
				expectedError: errors.New("test error"),
				expectedValue: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualError := (error)(nil)
			actualValue := (interface{})(nil)
			(&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment().ExecuteWorkflow(
				func(ctx workflow.Context) error {
					actualFuture := NewReadyFuture(ctx, testCase.input.value, testCase.input.err)
					actualError = actualFuture.Get(ctx, &actualValue)

					return nil
				},
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedValue, actualValue)
		})
	}
}
