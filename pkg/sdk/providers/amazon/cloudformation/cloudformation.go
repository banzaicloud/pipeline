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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
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
