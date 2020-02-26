// Copyright © 2018 Banzai Cloud
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

package oci

import "fmt"

// EntityNotFoundError specific error for not found entities
type EntityNotFoundError struct {
	Type string
	Id   string
}

func (e *EntityNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Type, e.Id)
}

// IsEntityNotFoundError returns false if the error is not EntityNotFoundError, otherwise true
func IsEntityNotFoundError(err error) (ok bool) {
	_, ok = err.(*EntityNotFoundError)
	return ok
}

type servicefailure struct {
	StatusCode int
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (se servicefailure) Error() string {
	return fmt.Sprintf("Service error:%s. %s. http status code: %d",
		se.Code, se.Message, se.StatusCode)
}

func (se servicefailure) GetHTTPStatusCode() int {
	return se.StatusCode
}

func (se servicefailure) GetMessage() string {
	return se.Message
}

func (se servicefailure) GetCode() string {
	return se.Code
}
