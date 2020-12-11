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
	"sort"
	"strconv"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/pkg/sdk/semver"
)

// decodeStackValue is a mapstructure decode function which parses a string
// value to the specified non-pointer type and returns the resulting value or an
// error on failure.
func decodeStackValue(inputType reflect.Type, outputType reflect.Type, inputValue interface{}) (interface{}, error) {
	inputValueString, isOk := inputValue.(string)
	if inputType.Kind() != reflect.String ||
		!isOk {
		return inputValue, nil
	}

	resultType := reflect.New(outputType).Elem().Interface() // Note: new adds an additional reference layer to the original type.

	return parseStackValue(inputValueString, resultType)
}

// NewOptionalStackParameter returns an initialized CloudFormation stack
// parameter. In case the passed condition is true, the passed value is used,
// otherwise the `UserPreviousValue` flag is set to true.
func NewOptionalStackParameter(key string, shouldUseNewValueInsteadOfPrevious bool, newValue string) (parameter *cloudformation.Parameter) {
	parameter = &cloudformation.Parameter{
		ParameterKey: aws.String(key),
	}

	if shouldUseNewValueInsteadOfPrevious {
		parameter.ParameterValue = aws.String(newValue)
	} else {
		parameter.UsePreviousValue = aws.Bool(true)
	}

	return parameter
}

// ParseStackParameters parses the specified outputs into the provided object
// pointer and returns an error on failure.
//
// The object pointer may be
//
// 1. an arbitrary struct parsing into the exported fields of the struct (use
// mapstructure tags if the fields are named differently than the output keys,
// parsing is case insensitive aside from the exported field requirement) and
// checking the existence of the corresponding fields among the outputs,
//
// 2. a map[string]string parsing all available parameter values into a map and
// checking the existence of preinitialized keys.
func ParseStackOutputs(outputs []*cloudformation.Output, objectPointer interface{}) (err error) {
	rawValues := make(map[string]string, len(outputs))
	for _, output := range outputs {
		rawValues[aws.StringValue(output.OutputKey)] = aws.StringValue(output.OutputValue)
	}

	return ParseStackValues(rawValues, objectPointer)
}

// ParseStackParameters parses the specified parameters into the provided object
// pointer and returns an error on failure.
//
// The object pointer may be
//
// 1. an arbitrary struct parsing into the exported fields of the struct (use
// mapstructure tags if the fields are named differently than the parameter
// keys, parsing is case insensitive aside from the exported field requirement)
// and checking the existence of the corresponding fields among the parameters,
//
// 2. a map[string]string parsing all available parameter values into a map and
// checking the existence of preinitialized keys.
func ParseStackParameters(parameters []*cloudformation.Parameter, objectPointer interface{}) (err error) {
	rawValues := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		rawValues[aws.StringValue(parameter.ParameterKey)] = aws.StringValue(parameter.ParameterValue)
	}

	return ParseStackValues(rawValues, objectPointer)
}

// parseStackValue parses a string value to the specified non-pointer
// type and returns the resulting value or an error on failure.
func parseStackValue(rawValue string, resultType interface{}) (result interface{}, err error) {
	switch typedPointer := (resultType).(type) {
	case bool:
		return strconv.ParseBool(rawValue)
	case float64:
		return strconv.ParseFloat(rawValue, 0)
	case int:
		return strconv.Atoi(rawValue)
	case semver.Version:
		return semver.NewVersionFromString(rawValue)
	case string:
		return rawValue, nil
	case uint:
		typedResult, err := strconv.ParseUint(rawValue, 10, 0)

		return uint(typedResult), err
	default:
		return nil, errors.New(fmt.Sprintf("string stack value parsing for type %T is not implemented", typedPointer))
	}
}

// ParseStackValues parses the specified raw values into the provided object
// pointer and returns an error on failure.
//
// The object pointer may be
//
// 1. an arbitrary struct parsing into the exported fields of the struct (use
// mapstructure tags if the fields are named differently than the parameter
// keys, parsing is case insensitive aside from the exported field requirement),
//
// 2. a map[string]string copying the raw values and checking the existence of
// the preinitialized keys.
func ParseStackValues(rawValues map[string]string, objectPointer interface{}) (err error) {
	if objectPointer == nil {
		return errors.New("object pointer is nil")
	}

	expectedKeysMap := make(map[string]interface{})
	err = mapstructure.Decode(objectPointer, &expectedKeysMap)
	if err != nil {
		return errors.WrapIf(err, "decoding expected keys failed")
	}

	expectedKeyErrors := make([]error, 0, len(expectedKeysMap))
	for expectedKey := range expectedKeysMap {
		if _, isExisting := rawValues[expectedKey]; !isExisting {
			expectedKeyErrors = append(expectedKeyErrors, errors.Errorf("missing expected key %s", expectedKey))
		}
	}
	if len(expectedKeyErrors) != 0 {
		sort.Slice(expectedKeyErrors, func(firstIndex, secondIndex int) (isLessThan bool) {
			return expectedKeyErrors[firstIndex].Error() < expectedKeyErrors[secondIndex].Error()
		})

		return errors.Combine(expectedKeyErrors...)
	}

	objectDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: decodeStackValue,
		Result:     objectPointer,
	})
	if err != nil {
		return errors.WrapIf(err, "initializing object decoder failed")
	}

	return objectDecoder.Decode(rawValues)
}
