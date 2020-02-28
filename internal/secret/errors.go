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

package secret

// ValidationError is returned when a request is semantically invalid.
type ValidationError struct {
	message    string
	violations []string
}

// NewValidationError returns a new ValidationError.
func NewValidationError(message string, violations []string) ValidationError {
	return ValidationError{
		message:    message,
		violations: violations,
	}
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.message != "" {
		return e.message
	}

	return "invalid request"
}

// Violations returns details of the failed validation.
func (e ValidationError) Violations() []string {
	return e.violations[:]
}

// Validation tells a client that this error is related to a semantic validation of the request.
// Can be used to translate the error to status codes for example.
func (ValidationError) Validation() bool {
	return true
}

// ServiceError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (ValidationError) ServiceError() bool {
	return true
}
