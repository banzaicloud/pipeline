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

package internal

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"go.uber.org/cadence"
)

// encodedError is a serialized version of the AWS SDK error.
type encodedError struct {
	Code    string
	Message string
	OrigErr string
}

// Error implements the error interface.
func (e encodedError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// EncodeError encodes an error returned by the AWS SDK as a Cadence error.
func EncodeError(err error) error {
	if err == nil {
		return nil
	}

	if aerr, ok := err.(awserr.Error); ok {
		newErr := encodedError{
			Code:    aerr.Code(),
			Message: aerr.Message(),
		}

		// TODO: try to encode the original error as well?
		if aerr.OrigErr() != nil {
			newErr.OrigErr = aerr.OrigErr().Error()
		}

		return cadence.NewCustomError("AWS_SDK_"+aerr.Code(), newErr)
	}

	return err
}
