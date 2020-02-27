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

// Strings holds strings in an array
type Strings struct {
	strings []string
}

// NodePoolOptions holds node pool options as Strings
type NodePoolOptions struct {
	Images             Strings
	KubernetesVersions Strings
	Shapes             Strings
}

// Has checks if the strings array has a value
func (s Strings) Has(value string) bool {
	for _, v := range s.strings {
		if v == value {
			return true
		}
	}

	return false
}

// Get gets the raw array
func (s Strings) Get() []string {
	return s.strings
}
