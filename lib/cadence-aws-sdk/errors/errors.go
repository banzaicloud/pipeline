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

package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"go.uber.org/cadence"
)

// decodedError is a serialized version of the AWS SDK error.
type decodedError struct {
	Code    string
	Message string
	OrigErr string
}

// decodedError implements the error interface.
func (e decodedError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// DecodeError decodes an error returned by a Cadence activity.
// If the error is an AWS SDK error, it gets converted to an AWS SDK compatible error interface.
// Otherwise the original error is returned.
func DecodeError(err error) error {
	if err == nil {
		return nil
	}

	customError, ok := err.(*cadence.CustomError)
	if !ok {
		return err
	}

	if !strings.HasPrefix(customError.Reason(), "AWS_SDK_") {
		return err
	}

	if !customError.HasDetails() {
		return awserr.New(
			strings.TrimPrefix(customError.Reason(), "AWS_SDK_"),
			customError.Error(),
			nil,
		)
	}

	var decErr decodedError

	if e := customError.Details(&decErr); e != nil {
		return err
	}

	return awserr.New(decErr.Code, decErr.Message, errors.New(decErr.OrigErr))
}
