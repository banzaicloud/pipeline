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

package semver

// Versions defines a version collection (slice) type for sorting purposes.
type Versions []Version

// Len returns the length of the collection.
func (versions Versions) Len() (length int) {
	return len(versions)
}

// Less returns true if the collection item behind the first specified index is
// less than the collection item behind the second provided index.
func (versions Versions) Less(firstIndex, secondIndex int) (isLessThan bool) {
	if versions == nil ||
		firstIndex < 0 ||
		firstIndex > len(versions) ||
		secondIndex < 0 ||
		secondIndex > len(versions) {
		return false
	}

	return versions[firstIndex].IsLessThan(versions[secondIndex])
}

// Swap replaces the collection items behind the specified indices with each other.
func (versions Versions) Swap(firstIndex, secondIndex int) {
	if versions == nil ||
		firstIndex < 0 ||
		firstIndex > len(versions) ||
		secondIndex < 0 ||
		secondIndex > len(versions) {
		return
	}

	versions[firstIndex], versions[secondIndex] = versions[secondIndex], versions[firstIndex]
}
