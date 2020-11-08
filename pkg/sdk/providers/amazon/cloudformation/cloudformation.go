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
	"sort"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/mitchellh/mapstructure"
)

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
// 2. a map[string]interface{} parsing all available outputs into the provided
// preinitialized types (or into string if the corresponding key is not
// preinitialized) and checking the existence of preinitialized keys, or
//
// 3. a map[string]string parsing all available parameter values into a map and
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
// 2. a map[string]interface{} parsing all available parameters into the
// provided preinitialized types (or into string if the corresponding key is not
// preinitialized) and checking the existence of preinitialized keys, or
//
// 3. a map[string]string parsing all available parameter values into a map and
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
	case string:
		return rawValue, nil
	case uint:
		var result uint64
		result, err = strconv.ParseUint(rawValue, 10, 0)

		return uint(result), err
	default:
		return nil, errors.New(fmt.Sprintf("parse string value type %T not implemented", typedPointer))
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
// 2. a map[string]interface{} parsing all available parameters into the
// provided preinitialized types (or into string if the corresponding key is not
// preinitialized), or
//
// 3. a map[string]string copying the raw values and checking the existence of
// the preinitialized keys.
func ParseStackValues(rawValues map[string]string, objectPointer interface{}) (err error) {
	objectPointerRepresentation := fmt.Sprintf("%v", objectPointer)
	if objectPointerRepresentation == "<nil>" { // Note: instead of == nil, https://golang.org/doc/faq#nil_error.
		return errors.New("object pointer is nil")
	} else if !strings.HasPrefix(objectPointerRepresentation, "&") {
		return errors.Errorf("invalid non-pointer object %s", objectPointerRepresentation)
	}

	objectMap := make(map[string]interface{})
	err = mapstructure.Decode(objectPointer, &objectMap)
	if err != nil {
		return errors.WrapIf(err, "decoding associative types from object pointer failed (struct or map is expected)")
	}

	parsedValues, err := parseStackValuesMap(rawValues, objectMap)
	if err != nil {
		return errors.Wrap(err, "parsing values failed")
	}

	return mapstructure.Decode(&parsedValues, objectPointer)
}

// parseStackValuesMap returns a strongly typed parsed value map by parsing the
// specified raw string values into a typed map based on the specified key and
// type map and returns an error on failure. Unexpected keys are parsed as
// string values.
func parseStackValuesMap(
	rawValues map[string]string,
	keysAndTypes map[string]interface{},
) (parsedValues map[string]interface{}, err error) {
	if rawValues == nil {
		return nil, errors.New("raw value map is nil")
	} else if keysAndTypes == nil {
		return nil, errors.New("keys and types map is nil")
	}

	parseErrors := make([]error, 0)
	// Note: valueMap is the actual state while parsedValues is the desired
	// state. They are kept separate for checking missing values purposes.
	parsedValues = make(map[string]interface{}, len(rawValues))
	for key, rawValue := range rawValues {
		if _, isExisting := keysAndTypes[key]; !isExisting {
			keysAndTypes[key] = "" // Note: using string for unknown parameter types.
		}

		parsedValues[key], err = parseStackValue(rawValue, keysAndTypes[key])
		if err != nil {
			parseErrors = append(
				parseErrors,
				errors.Wrapf(err, "parsing %s value %s failed", key, rawValue),
			)
		}
	}

	for key := range keysAndTypes {
		if _, isExisting := parsedValues[key]; !isExisting {
			parseErrors = append(parseErrors, errors.New(fmt.Sprintf("missing requested value %s", key)))
		}
	}

	if len(parseErrors) != 0 {
		// Note: making combined errors deterministic for better experience and
		// testability.
		sort.Slice(parseErrors, func(firstIndex, secondIndex int) (isLessThan bool) {
			return parseErrors[firstIndex].Error() < parseErrors[secondIndex].Error()
		})

		return nil, errors.Combine(parseErrors...)
	}

	return parsedValues, nil
}
