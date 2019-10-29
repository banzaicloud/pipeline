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

package monitoring

import "fmt"

type requiredFieldError struct {
	fieldName string
}

type invalidIngressHostError struct {
	hostType string
}

type cannotDisabledError struct {
	fieldName string
}

func (e invalidIngressHostError) Error() string {
	return fmt.Sprintf("invalid %s ingress host", e.hostType)
}

func (e requiredFieldError) Error() string {
	return fmt.Sprintf("%q cannot be empty", e.fieldName)
}

func (e cannotDisabledError) Error() string {
	return fmt.Sprintf("%s cannot be disabled", e.fieldName)
}
