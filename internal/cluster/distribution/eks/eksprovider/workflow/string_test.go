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

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndexStrings(t *testing.T) {
	type inputType struct {
		candidate  string
		collection []string
	}

	testCases := []struct {
		caseDescription string
		expectedIndex   int
		input           inputType
	}{
		{
			caseDescription: "found first index success",
			expectedIndex:   0,
			input: inputType{
				candidate:  "candidate",
				collection: []string{"candidate", "1", "2"},
			},
		},
		{
			caseDescription: "found middle index success",
			expectedIndex:   1,
			input: inputType{
				candidate:  "candidate",
				collection: []string{"0", "candidate", "2"},
			},
		},
		{
			caseDescription: "found last index success",
			expectedIndex:   2,
			input: inputType{
				candidate:  "candidate",
				collection: []string{"0", "1", "candidate"},
			},
		},
		{
			caseDescription: "not found index success",
			expectedIndex:   -1,
			input: inputType{
				candidate:  "candidate",
				collection: []string{"0", "1", "2"},
			},
		},
		{
			caseDescription: "nil collection success",
			expectedIndex:   -1,
			input: inputType{
				candidate:  "candidate",
				collection: nil,
			},
		},
		{
			caseDescription: "empty collection success",
			expectedIndex:   -1,
			input: inputType{
				candidate:  "candidate",
				collection: []string{},
			},
		},
		{
			caseDescription: "not found empty candidate success",
			expectedIndex:   -1,
			input: inputType{
				candidate:  "",
				collection: []string{"0", "1", "2"},
			},
		},
		{
			caseDescription: "found empty candidate success",
			expectedIndex:   0,
			input: inputType{
				candidate:  "",
				collection: []string{"", "1", "2"},
			},
		},
		{
			caseDescription: "nil collection, empty candidate success",
			expectedIndex:   -1,
			input: inputType{
				candidate:  "",
				collection: nil,
			},
		},
		{
			caseDescription: "empty collection, empty candidate success",
			expectedIndex:   -1,
			input: inputType{
				candidate:  "",
				collection: []string{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIndex := indexStrings(testCase.input.collection, testCase.input.candidate)

			require.Equal(t, testCase.expectedIndex, actualIndex)
		})
	}
}
