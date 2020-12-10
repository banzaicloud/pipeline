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

func TestCompareInts(t *testing.T) {
	type inputType struct {
		first  int
		second int
	}

	testCases := []struct {
		caseDescription string
		expectedResult  Compared
		input           inputType
	}{
		{
			caseDescription: "first < second -> successful ComparedLess",
			expectedResult:  ComparedLess,
			input: inputType{
				first:  0,
				second: 1,
			},
		},
		{
			caseDescription: "first == second -> successful ComparedEqual",
			expectedResult:  ComparedEqual,
			input: inputType{
				first:  2,
				second: 2,
			},
		},
		{
			caseDescription: "first > second -> successful ComparedGreater",
			expectedResult:  ComparedGreater,
			input: inputType{
				first:  4,
				second: 3,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualResult := CompareInts(testCase.input.first, testCase.input.second)

			require.Equal(t, testCase.expectedResult, actualResult)
		})
	}
}

func TestCompareStrings(t *testing.T) {
	type inputType struct {
		first  string
		second string
	}

	testCases := []struct {
		caseDescription string
		expectedResult  Compared
		input           inputType
	}{
		{
			caseDescription: "first < second -> successful ComparedLess",
			expectedResult:  ComparedLess,
			input: inputType{
				first:  "0",
				second: "1",
			},
		},
		{
			caseDescription: "first == second -> successful ComparedEqual",
			expectedResult:  ComparedEqual,
			input: inputType{
				first:  "2",
				second: "2",
			},
		},
		{
			caseDescription: "first > second -> successful ComparedGreater",
			expectedResult:  ComparedGreater,
			input: inputType{
				first:  "4",
				second: "3",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualResult := CompareStrings(testCase.input.first, testCase.input.second)

			require.Equal(t, testCase.expectedResult, actualResult)
		})
	}
}
