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
