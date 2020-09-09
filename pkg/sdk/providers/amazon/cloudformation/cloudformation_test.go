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
	"fmt"
	"testing"

	"emperror.dev/errors"
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

func TestParseStackParameters(t *testing.T) {
	type inputType struct {
		objectPointer interface{}
		parameters    []*cloudformation.Parameter
	}

	type outputType struct {
		expectedError         error
		expectedObjectPointer interface{}
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "non-pointer object error",
			input: inputType{
				objectPointer: struct{}{},
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError:         errors.New("invalid non-pointer object '{}'"),
				expectedObjectPointer: struct{}{},
			},
		},
		{
			caseName: "nil object pointer error",
			input: inputType{
				objectPointer: (*map[string]interface{})(nil),
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError:         errors.New("invalid nil object pointer"),
				expectedObjectPointer: (*map[string]interface{})(nil),
			},
		},
		{
			caseName: "invalid non-associative object behind pointer error",
			input: inputType{
				objectPointer: &[]interface{}{},
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError: errors.New(
					"decoding associative types from object pointer failed (struct or map is expected)" +
						": '' expected a map, got 'slice'",
				),
				expectedObjectPointer: &[]interface{}{},
			},
		},
		{
			caseName: "invalid non-associative object pointer decoding error",
			input: inputType{
				objectPointer: &map[string]interface{}{
					"Float": false,
					"Int":   false,
				},
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError: errors.New(
					"parsing cloudformation stack parameter failed: strconv.ParseBool: parsing \"0.5\": invalid syntax" +
						"; parsing cloudformation stack parameter failed: strconv.ParseBool: parsing \"5\": invalid syntax",
				),
				expectedObjectPointer: &map[string]interface{}{
					"Float": false,
					"Int":   false,
				},
			},
		},
		{
			caseName: "missing requested parameter error",
			input: inputType{
				objectPointer: &map[string]interface{}{
					"NotExistingKey": false,
				},
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError: errors.New("missing requested parameter 'NotExistingKey'"),
				expectedObjectPointer: &map[string]interface{}{
					"NotExistingKey": false,
				},
			},
		},
		{
			caseName: "map[string]interface{} success",
			input: inputType{
				objectPointer: &map[string]interface{}{
					"Bool":   false,
					"Float":  0.0,
					"Int":    0,
					"String": "",
				},
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Extra"),
						ParameterValue: aws.String("parameter"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &map[string]interface{}{
					"Bool":   true,
					"Extra":  "parameter",
					"Float":  0.5,
					"Int":    5,
					"String": "value",
				},
			},
		},
		{
			caseName: "struct success",
			input: inputType{
				objectPointer: &struct {
					Bool   bool
					Float  float64
					Int    int
					String string
				}{},
				parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("Bool"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("Extra"),
						ParameterValue: aws.String("parameter"),
					},
					{
						ParameterKey:   aws.String("Float"),
						ParameterValue: aws.String("0.5"),
					},
					{
						ParameterKey:   aws.String("Int"),
						ParameterValue: aws.String("5"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &struct {
					Bool   bool
					Float  float64
					Int    int
					String string
				}{
					Bool:   true,
					Float:  0.5,
					Int:    5,
					String: "value",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualError := ParseStackParameters(testCase.input.parameters, testCase.input.objectPointer)

			if testCase.output.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedObjectPointer, testCase.input.objectPointer)
		})
	}
}

func TestParseStackParameterValue(t *testing.T) {
	type inputType struct {
		rawValue   string
		resultType interface{}
	}

	type outputType struct {
		expectedError  error
		expectedResult interface{}
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "invalid bool value error",
			input: inputType{
				rawValue:   "value",
				resultType: false,
			},
			output: outputType{
				expectedError:  errors.New("strconv.ParseBool: parsing \"value\": invalid syntax"),
				expectedResult: false,
			},
		},
		{
			caseName: "valid bool value success",
			input: inputType{
				rawValue:   "true",
				resultType: false,
			},
			output: outputType{
				expectedError:  nil,
				expectedResult: true,
			},
		},
		{
			caseName: "invalid float64 value error",
			input: inputType{
				rawValue:   "value",
				resultType: 0.0,
			},
			output: outputType{
				expectedError:  errors.New("strconv.ParseFloat: parsing \"value\": invalid syntax"),
				expectedResult: 0.0,
			},
		},
		{
			caseName: "valid float64 value success",
			input: inputType{
				rawValue:   "5.4",
				resultType: 0.0,
			},
			output: outputType{
				expectedError:  nil,
				expectedResult: 5.4,
			},
		},
		{
			caseName: "invalid int value error",
			input: inputType{
				rawValue:   "value",
				resultType: 0,
			},
			output: outputType{
				expectedError:  errors.New("strconv.Atoi: parsing \"value\": invalid syntax"),
				expectedResult: 0,
			},
		},
		{
			caseName: "valid int value success",
			input: inputType{
				rawValue:   "5",
				resultType: 0,
			},
			output: outputType{
				expectedError:  nil,
				expectedResult: 5,
			},
		},
		{
			caseName: "valid string value success",
			input: inputType{
				rawValue:   "value",
				resultType: "",
			},
			output: outputType{
				expectedError:  nil,
				expectedResult: "value",
			},
		},
		{
			caseName: "invalid uint value error",
			input: inputType{
				rawValue:   "value",
				resultType: uint(0),
			},
			output: outputType{
				expectedError:  errors.New("strconv.ParseUint: parsing \"value\": invalid syntax"),
				expectedResult: uint(0),
			},
		},
		{
			caseName: "valid uint value success",
			input: inputType{
				rawValue:   "5",
				resultType: uint(0),
			},
			output: outputType{
				expectedError:  nil,
				expectedResult: uint(5),
			},
		},
		{
			caseName: "unimplemented type pointer error",
			input: inputType{
				rawValue:   "true",
				resultType: struct{}{},
			},
			output: outputType{
				expectedError:  errors.New(fmt.Sprintf("parse string value type %T not implemented", struct{}{})),
				expectedResult: nil,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualResult, actualError := parseStackParameterValue(testCase.input.rawValue, testCase.input.resultType)

			if testCase.output.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedResult, actualResult)
		})
	}
}
