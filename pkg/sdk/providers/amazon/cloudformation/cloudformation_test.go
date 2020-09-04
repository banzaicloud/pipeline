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

package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/require"
)

func TestNewOptionalStackParameter(t *testing.T) {
	type inputType struct {
		key                                string
		newValue                           string
		shouldUseNewValueInsteadOfPrevious bool
	}

	type outputType struct {
		expectedParameter *cloudformation.Parameter
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "not set value",
			input: inputType{
				key:                                "key",
				newValue:                           "value",
				shouldUseNewValueInsteadOfPrevious: false,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:     aws.String("key"),
					UsePreviousValue: aws.Bool(true),
				},
			},
		},
		{
			caseName: "set value",
			input: inputType{
				key:                                "key",
				newValue:                           "value",
				shouldUseNewValueInsteadOfPrevious: true,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:   aws.String("key"),
					ParameterValue: aws.String("value"),
				},
			},
		},
		{
			caseName: "empty key, empty new value, false should use new value instead of previous",
			input: inputType{
				key:                                "",
				newValue:                           "",
				shouldUseNewValueInsteadOfPrevious: false,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:     aws.String(""),
					UsePreviousValue: aws.Bool(true),
				},
			},
		},
		{
			caseName: "empty key, empty new value, true should use new value instead of previous",
			input: inputType{
				key:                                "",
				newValue:                           "",
				shouldUseNewValueInsteadOfPrevious: true,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:   aws.String(""),
					ParameterValue: aws.String(""),
				},
			},
		},
		{
			caseName: "empty key, not empty new value, false should use new value instead of previous",
			input: inputType{
				key:                                "",
				newValue:                           "value",
				shouldUseNewValueInsteadOfPrevious: false,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:     aws.String(""),
					UsePreviousValue: aws.Bool(true),
				},
			},
		},
		{
			caseName: "empty key, not empty new value, true should use new value instead of previous",
			input: inputType{
				key:                                "",
				newValue:                           "value",
				shouldUseNewValueInsteadOfPrevious: true,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:   aws.String(""),
					ParameterValue: aws.String("value"),
				},
			},
		},
		{
			caseName: "not empty key, empty new value, false should use new value instead of previous",
			input: inputType{
				key:                                "key",
				newValue:                           "",
				shouldUseNewValueInsteadOfPrevious: false,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:     aws.String("key"),
					UsePreviousValue: aws.Bool(true),
				},
			},
		},
		{
			caseName: "not empty key, empty new value, true should use new value instead of previous",
			input: inputType{
				key:                                "key",
				newValue:                           "",
				shouldUseNewValueInsteadOfPrevious: true,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:   aws.String("key"),
					ParameterValue: aws.String(""),
				},
			},
		},
		{
			caseName: "not empty key, not empty new value, false should use new value instead of previous",
			input: inputType{
				key:                                "key",
				newValue:                           "value",
				shouldUseNewValueInsteadOfPrevious: false,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:     aws.String("key"),
					UsePreviousValue: aws.Bool(true),
				},
			},
		},
		{
			caseName: "not empty key, not empty new value, true should use new value instead of previous",
			input: inputType{
				key:                                "key",
				newValue:                           "value",
				shouldUseNewValueInsteadOfPrevious: true,
			},
			output: outputType{
				expectedParameter: &cloudformation.Parameter{
					ParameterKey:   aws.String("key"),
					ParameterValue: aws.String("value"),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualParameter := NewOptionalStackParameter(
				testCase.input.key,
				testCase.input.shouldUseNewValueInsteadOfPrevious,
				testCase.input.newValue,
			)

			require.Equal(t, testCase.output.expectedParameter, actualParameter)
		})
	}
}
