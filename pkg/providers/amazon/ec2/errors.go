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

package ec2

import "fmt"

type errSpotRequestFailed struct {
	Code string
}

// NewSpotRequestFailedError creates a new errSpotRequestFailed
func NewSpotRequestFailedError(code string) error {
	return errSpotRequestFailed{
		Code: code,
	}
}

func (e errSpotRequestFailed) Error() string          { return fmt.Sprintf("spot request failed: %s", e.Code) }
func (e errSpotRequestFailed) IsFinal() bool          { return true }
func (e errSpotRequestFailed) Context() []interface{} { return []interface{}{"code", e.Code} }
