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

const (
	// ComparedLess is the comparison result describing the receiver is less
	// than the argument.
	ComparedLess = Compared(-1)

	// ComparedEqual is the comparison result describing the receiver is equal
	// to the argument.
	ComparedEqual = Compared(0)

	// ComparedGreater is the comparison result describing the receiver is
	// greater than the argument.
	ComparedGreater = Compared(1)
)

// Compared describes pseudo-enum results of version comparisons.
type Compared int

// CompareIns compares two int values and returns the result as a Compared
// value.
func CompareInts(first, second int) (result Compared) {
	if first < second {
		return ComparedLess
	} else if first > second {
		return ComparedGreater
	}

	return ComparedEqual
}

// CompareStrings compares two string values and returns the result as a
// Compared value.
func CompareStrings(first, second string) (result Compared) {
	if first < second {
		return ComparedLess
	} else if first > second {
		return ComparedGreater
	}

	return ComparedEqual
}
