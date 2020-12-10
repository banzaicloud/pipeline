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

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionsLen(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedLength  int
		inputVersions   Versions
	}{
		{
			caseDescription: "nil versions -> 0 success",
			expectedLength:  0,
			inputVersions:   nil,
		},
		{
			caseDescription: "empty versions -> 0 success",
			expectedLength:  0,
			inputVersions:   Versions([]Version{}),
		},
		{
			caseDescription: "not empty versions -> len(Versions) success",
			expectedLength:  3,
			inputVersions:   Versions([]Version{"1.2.3", "4.5.6", "7.8.9"}),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualLength := testCase.inputVersions.Len()

			require.Equal(t, testCase.expectedLength, actualLength)
		})
	}
}

func TestVersionsLess(t *testing.T) {
	type inputType struct {
		versions    Versions
		firstIndex  int
		secondIndex int
	}

	testCases := []struct {
		caseDescription    string
		expectedIsLessThan bool
		input              inputType
	}{
		{
			caseDescription:    "nil versions -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    nil,
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "empty versions -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{}),
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "less -> true success",
			expectedIsLessThan: true,
			input: inputType{
				versions:    Versions([]Version{"1.2.3", "4.5.6"}),
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "equal -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{"1.2.3", "1.2.3"}),
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "greater -> true success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "first index < 0 -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  -1,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "first index > len(versions) -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  999,
				secondIndex: 1,
			},
		},
		{
			caseDescription:    "second index < 0 -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  0,
				secondIndex: -1,
			},
		},
		{
			caseDescription:    "second index > len(versions) -> false success",
			expectedIsLessThan: false,
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  0,
				secondIndex: 999,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIsLessThan := testCase.input.versions.Less(testCase.input.firstIndex, testCase.input.secondIndex)

			require.Equal(t, testCase.expectedIsLessThan, actualIsLessThan)
		})
	}
}

func TestVersionsSwap(t *testing.T) {
	type inputType struct {
		versions    Versions
		firstIndex  int
		secondIndex int
	}

	testCases := []struct {
		caseDescription  string
		expectedVersions Versions
		input            inputType
	}{
		{
			caseDescription:  "nil versions -> original versions",
			expectedVersions: nil,
			input: inputType{
				versions:    nil,
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:  "empty versions -> original versions",
			expectedVersions: Versions([]Version{}),
			input: inputType{
				versions:    Versions([]Version{}),
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:  "not empty versions -> swapped success",
			expectedVersions: Versions([]Version{"4.5.6", "1.2.3"}),
			input: inputType{
				versions:    Versions([]Version{"1.2.3", "4.5.6"}),
				firstIndex:  0,
				secondIndex: 1,
			},
		},
		{
			caseDescription:  "first index < 0 -> original versions",
			expectedVersions: Versions([]Version{"7.8.9", "4.5.6"}),
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  -1,
				secondIndex: 1,
			},
		},
		{
			caseDescription:  "first index > len(versions) -> original versions",
			expectedVersions: Versions([]Version{"7.8.9", "4.5.6"}),
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  999,
				secondIndex: 1,
			},
		},
		{
			caseDescription:  "second index < 0 -> original versions",
			expectedVersions: Versions([]Version{"7.8.9", "4.5.6"}),
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  0,
				secondIndex: -1,
			},
		},
		{
			caseDescription:  "second index > len(versions) -> original versions",
			expectedVersions: Versions([]Version{"7.8.9", "4.5.6"}),
			input: inputType{
				versions:    Versions([]Version{"7.8.9", "4.5.6"}),
				firstIndex:  0,
				secondIndex: 999,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			testCase.input.versions.Swap(testCase.input.firstIndex, testCase.input.secondIndex)

			require.Equal(t, testCase.expectedVersions, testCase.input.versions)
		})
	}
}
