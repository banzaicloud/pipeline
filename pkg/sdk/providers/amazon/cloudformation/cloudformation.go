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

// ParseStackParameters parses the specified parameters into the provided object
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
// 3. a map[string]string to extract all available parameter values in a map.
func ParseStackParameters(parameters []*cloudformation.Parameter, objectPointer interface{}) (err error) {
	objectPointerRepresentation := fmt.Sprintf("%v", objectPointer)
	if objectPointerRepresentation == "<nil>" { // Note: instead of == nil, https://golang.org/doc/faq#nil_error.
		return errors.New("invalid nil object pointer")
	} else if !strings.HasPrefix(objectPointerRepresentation, "&") {
		return errors.New(fmt.Sprintf("invalid non-pointer object '%s'", objectPointerRepresentation))
	}

	objectMap := make(map[string]interface{})
	err = mapstructure.Decode(objectPointer, &objectMap)
	if err != nil {
		return errors.WrapIf(err, "decoding associative types from object pointer failed (struct or map is expected)")
	}

	// Note: parameterMap is the actual state while objectMap is the desired
	// state. They are kept separate for requested parameter checking purposes.
	parameterMap := make(map[string]interface{}, len(parameters))
	parseErrors := make([]error, 0)
	for _, parameter := range parameters {
		parameterKey := aws.StringValue(parameter.ParameterKey)
		if _, isExisting := objectMap[parameterKey]; !isExisting {
			objectMap[parameterKey] = "" // Note: using string for unknown parameter types.
		}

		parameterValue := aws.StringValue(parameter.ParameterValue)
		parameterMap[parameterKey], err = parseStackParameterValue(parameterValue, objectMap[parameterKey])
		if err != nil {
			parseErrors = append(parseErrors, errors.WrapWithDetails(
				err,
				"parsing cloudformation stack parameter failed",
				"parameterKey", parameterKey,
				"parameterValue", parameterValue,
			))
		}
	}

	for objectKey := range objectMap {
		if _, isExisting := parameterMap[objectKey]; !isExisting {
			parseErrors = append(parseErrors, errors.New(fmt.Sprintf("missing requested parameter '%s'", objectKey)))
		}
	}

	if len(parseErrors) != 0 {
		// Note: making combined errors deterministic for better experience and
		// testability.
		sort.Slice(parseErrors, func(firstIndex, secondIndex int) (isLessThan bool) {
			return parseErrors[firstIndex].Error() < parseErrors[secondIndex].Error()
		})

		return errors.Combine(parseErrors...)
	}

	return mapstructure.Decode(&parameterMap, objectPointer)
}

// parseStackParameterValue parses a string value to the specified non-pointer
// type and returns the resulting value or an error on failure.
func parseStackParameterValue(rawValue string, resultType interface{}) (result interface{}, err error) {
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
