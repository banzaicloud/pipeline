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

// DetailedError encapsulates an error with a reason and one or more details.
type DetailedError interface {
	// error extends the interface with the standard error methods.
	error

	// Details returns the resulting (possibly nil) error after trying to decode
	// the error details into the provided value pointer.
	Details(valuePointers ...interface{}) (err error)
}
