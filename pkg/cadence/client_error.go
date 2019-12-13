// Copyright © 2019 Banzai Cloud
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
	"fmt"

	"emperror.dev/errors"
	"go.uber.org/cadence"
)

// Cadence client error reason.
const ClientErrorReason = "ClientError"

// NewClientError returns a new client error.
func NewClientError(err error) error {
	return cadence.NewCustomError(ClientErrorReason, err.Error())
}

// WrapClientError wraps an error into a custom cadence error if it's a client error.
func WrapClientError(err error) error {
	if err != nil {
		var clientError interface{ ClientError() bool }
		if errors.As(err, &clientError) && clientError.ClientError() {
			return NewClientError(err)
		}
	}

	return err
}

// UnwrapError returns a new detailed error based on the wrapped string in a ClientError,
// or the original error otherwise
func UnwrapError(err error) error {
	if detailser, ok := err.(interface{ Details(d ...interface{}) error }); ok {
		var str string
		if detailser.Details(&str) == nil {
			return fmt.Errorf("client error: %s", str)
		}
	}
	return err
}
