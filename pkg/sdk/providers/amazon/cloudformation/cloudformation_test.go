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
	"reflect"
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/pkg/sdk/semver"
)

func TestDecodeStackValue(t *testing.T) {
	type inputType struct {
		inputType  reflect.Type
		outputType reflect.Type
		inputValue interface{}
	}

	type outputType struct {
		expectedOutputValue interface{}
		expectedErr         error
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "not a string stack value input -> early exit success",
			input: inputType{
				inputType:  reflect.TypeOf(0),
				outputType: reflect.TypeOf(""),
				inputValue: 5,
			},
			output: outputType{
				expectedOutputValue: 5,
				expectedErr:         nil,
			},
		},
		{
			caseDescription: "int success -> success",
			input: inputType{
				inputType:  reflect.TypeOf(""),
				outputType: reflect.TypeOf(0),
				inputValue: "5",
			},
			output: outputType{
				expectedOutputValue: 5,
				expectedErr:         nil,
			},
		},
		{
			caseDescription: "int error -> error",
			input: inputType{
				inputType:  reflect.TypeOf(""),
				outputType: reflect.TypeOf(0),
				inputValue: "value",
			},
			output: outputType{
				expectedOutputValue: 0,
				expectedErr:         errors.New("strconv.Atoi: parsing \"value\": invalid syntax"),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualOutputValue, actualErr := decodeStackValue(
				testCase.input.inputType,
				testCase.input.outputType,
				testCase.input.inputValue,
			)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedOutputValue, actualOutputValue)
		})
	}
}

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

func TestParseStackOutputs(t *testing.T) {
	type inputType struct {
		objectPointer interface{}
		outputs       []*cloudformation.Output
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
			caseName: "map[string]string -> success",
			input: inputType{
				objectPointer: &map[string]string{
					"Bool":   "",
					"Float":  "",
					"Int":    "",
					"String": "",
					"Uint":   "",
				},
				outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Bool"),
						OutputValue: aws.String("true"),
					},
					{
						OutputKey:   aws.String("Extra"),
						OutputValue: aws.String("output"),
					},
					{
						OutputKey:   aws.String("Float"),
						OutputValue: aws.String("0.5"),
					},
					{
						OutputKey:   aws.String("Int"),
						OutputValue: aws.String("-5"),
					},
					{
						OutputKey:   aws.String("SemanticVersion"),
						OutputValue: aws.String("1.2.3-dev.4"),
					},
					{
						OutputKey:   aws.String("String"),
						OutputValue: aws.String("value"),
					},
					{
						OutputKey:   aws.String("Uint"),
						OutputValue: aws.String("5"),
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &map[string]string{
					"Bool":            "true",
					"Extra":           "output",
					"Float":           "0.5",
					"Int":             "-5",
					"SemanticVersion": "1.2.3-dev.4",
					"String":          "value",
					"Uint":            "5",
				},
			},
		},
		{
			caseName: "struct -> success",
			input: inputType{
				objectPointer: &struct {
					Bool            bool
					Float           float64
					Int             int
					SemanticVersion semver.Version
					String          string
					Uint            uint
				}{},
				outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Bool"),
						OutputValue: aws.String("true"),
					},
					{
						OutputKey:   aws.String("Extra"),
						OutputValue: aws.String("output"),
					},
					{
						OutputKey:   aws.String("Float"),
						OutputValue: aws.String("0.5"),
					},
					{
						OutputKey:   aws.String("Int"),
						OutputValue: aws.String("-5"),
					},
					{
						OutputKey:   aws.String("SemanticVersion"),
						OutputValue: aws.String("1.2.3-dev.4"),
					},
					{
						OutputKey:   aws.String("String"),
						OutputValue: aws.String("value"),
					},
					{
						OutputKey:   aws.String("Uint"),
						OutputValue: aws.String("5"),
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &struct {
					Bool            bool
					Float           float64
					Int             int
					SemanticVersion semver.Version
					String          string
					Uint            uint
				}{
					Bool:            true,
					Float:           0.5,
					Int:             -5,
					SemanticVersion: semver.NewVersionFromStringOrPanic("1.2.3-dev.4"),
					String:          "value",
					Uint:            5,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualError := ParseStackOutputs(testCase.input.outputs, testCase.input.objectPointer)

			if testCase.output.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedObjectPointer, testCase.input.objectPointer)
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
			caseName: "map[string]string -> success",
			input: inputType{
				objectPointer: &map[string]string{
					"Bool":   "",
					"Float":  "",
					"Int":    "",
					"String": "",
					"Uint":   "",
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
						ParameterValue: aws.String("-5"),
					},
					{
						ParameterKey:   aws.String("SemanticVersion"),
						ParameterValue: aws.String("1.2.3-dev.4"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
					{
						ParameterKey:   aws.String("Uint"),
						ParameterValue: aws.String("5"),
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &map[string]string{
					"Bool":            "true",
					"Extra":           "parameter",
					"Float":           "0.5",
					"Int":             "-5",
					"SemanticVersion": "1.2.3-dev.4",
					"String":          "value",
					"Uint":            "5",
				},
			},
		},
		{
			caseName: "struct -> success",
			input: inputType{
				objectPointer: &struct {
					Bool            bool
					Float           float64
					Int             int
					SemanticVersion semver.Version
					String          string
					Uint            uint
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
						ParameterValue: aws.String("-5"),
					},
					{
						ParameterKey:   aws.String("SemanticVersion"),
						ParameterValue: aws.String("1.2.3-dev.4"),
					},
					{
						ParameterKey:   aws.String("String"),
						ParameterValue: aws.String("value"),
					},
					{
						ParameterKey:   aws.String("Uint"),
						ParameterValue: aws.String("5"),
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &struct {
					Bool            bool
					Float           float64
					Int             int
					SemanticVersion semver.Version
					String          string
					Uint            uint
				}{
					Bool:            true,
					Float:           0.5,
					Int:             -5,
					SemanticVersion: semver.NewVersionFromStringOrPanic("1.2.3-dev.4"),
					String:          "value",
					Uint:            5,
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

func TestParseStackValue(t *testing.T) {
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
			caseName: "invalid semver.Version value error",
			input: inputType{
				rawValue:   "version-value",
				resultType: semver.Version(""),
			},
			output: outputType{
				expectedError:  errors.New("invalid version version-value"),
				expectedResult: semver.Version(""),
			},
		},
		{
			caseName: "valid semver.Version value success",
			input: inputType{
				rawValue:   "1.2.3-dev.4",
				resultType: semver.Version(""),
			},
			output: outputType{
				expectedError:  nil,
				expectedResult: semver.NewVersionFromStringOrPanic("1.2.3-dev.4"),
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
				expectedError: errors.New(
					fmt.Sprintf("string stack value parsing for type %T is not implemented", struct{}{}),
				),
				expectedResult: nil,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualResult, actualError := parseStackValue(testCase.input.rawValue, testCase.input.resultType)

			if testCase.output.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedResult, actualResult)
		})
	}
}

func TestParseStackValues(t *testing.T) {
	type inputType struct {
		rawValues     map[string]string
		objectPointer interface{}
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
			caseName: "nil object pointer -> error",
			input: inputType{
				rawValues:     map[string]string{},
				objectPointer: nil,
			},
			output: outputType{
				expectedError:         errors.New("object pointer is nil"),
				expectedObjectPointer: nil,
			},
		},
		{
			caseName: "expected keys decoding error -> error",
			input: inputType{
				rawValues:     map[string]string{},
				objectPointer: &[]struct{}{},
			},
			output: outputType{
				expectedError:         errors.New("decoding expected keys failed: '' expected a map, got 'slice'"),
				expectedObjectPointer: &[]struct{}{},
			},
		},
		{
			caseName: "missing expected key error -> error",
			input: inputType{
				rawValues: map[string]string{},
				objectPointer: &map[string]string{
					"AnotherExpectedKey": "",
					"ExpectedKey":        "",
				},
			},
			output: outputType{
				expectedError: errors.New("missing expected key AnotherExpectedKey; missing expected key ExpectedKey"),
				expectedObjectPointer: &map[string]string{
					"AnotherExpectedKey": "",
					"ExpectedKey":        "",
				},
			},
		},
		{
			caseName: "non-pointer object pointer -> error",
			input: inputType{
				rawValues: map[string]string{
					"error": "error",
				},
				objectPointer: struct{}{},
			},
			output: outputType{
				expectedError:         errors.New("initializing object decoder failed: result must be a pointer"),
				expectedObjectPointer: struct{}{},
			},
		},
		{
			caseName: "parsing error -> error",
			input: inputType{
				rawValues: map[string]string{
					"Error": "error",
				},
				objectPointer: &struct {
					Error struct{}
				}{},
			},
			output: outputType{
				expectedError: errors.New(
					"1 error(s) decoding:\n\n* error decoding 'Error'" +
						": string stack value parsing for type struct {} is not implemented",
				),
				expectedObjectPointer: &struct {
					Error struct{}
				}{},
			},
		},
		{
			caseName: "map[string]string -> success",
			input: inputType{
				rawValues: map[string]string{
					"Bool":            "true",
					"Extra":           "parameter",
					"Float":           "0.5",
					"Int":             "-5",
					"SemanticVersion": "1.2.3-dev.4",
					"String":          "value",
					"Uint":            "5",
				},
				objectPointer: &map[string]string{
					"Bool":            "",
					"Float":           "",
					"Int":             "",
					"SemanticVersion": "",
					"String":          "",
					"Uint":            "",
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &map[string]string{
					"Bool":            "true",
					"Extra":           "parameter",
					"Float":           "0.5",
					"Int":             "-5",
					"SemanticVersion": "1.2.3-dev.4",
					"String":          "value",
					"Uint":            "5",
				},
			},
		},
		{
			caseName: "struct -> success",
			input: inputType{
				rawValues: map[string]string{
					"Bool":            "true",
					"Extra":           "parameter",
					"Float":           "0.5",
					"Int":             "-5",
					"SemanticVersion": "1.2.3-dev.4",
					"String":          "value",
					"Uint":            "5",
				},
				objectPointer: &struct {
					Bool            bool
					Float           float64
					Int             int
					SemanticVersion semver.Version
					String          string
					Uint            uint
				}{
					Bool:            false,
					Float:           0.0,
					Int:             0,
					SemanticVersion: semver.NewVersionFromStringOrPanic("0.0.0"),
					String:          "",
					Uint:            uint(0),
				},
			},
			output: outputType{
				expectedError: nil,
				expectedObjectPointer: &struct {
					Bool            bool
					Float           float64
					Int             int
					SemanticVersion semver.Version
					String          string
					Uint            uint
				}{
					Bool:            true,
					Float:           0.5,
					Int:             -5,
					SemanticVersion: semver.NewVersionFromStringOrPanic("1.2.3-dev.4"),
					String:          "value",
					Uint:            uint(5),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualError := ParseStackValues(testCase.input.rawValues, testCase.input.objectPointer)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedObjectPointer, testCase.input.objectPointer)
		})
	}
}
