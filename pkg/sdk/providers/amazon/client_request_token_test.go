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

package amazon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNormalizedClientRequestToken(t *testing.T) {
	type inputType struct {
		elements []string
	}

	type outputType struct {
		expectedNormalizedClientRequestToken string
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "nil elements success",
			input: inputType{
				elements: nil,
			},
			output: outputType{
				expectedNormalizedClientRequestToken: "",
			},
		},
		{
			caseName: "single element success",
			input: inputType{
				elements: []string{
					"element",
				},
			},
			output: outputType{
				expectedNormalizedClientRequestToken: "element",
			},
		},
		{
			caseName: "multiple elements success",
			input: inputType{
				elements: []string{
					"multiple",
					"token",
					"elements",
				},
			},
			output: outputType{
				expectedNormalizedClientRequestToken: "multiple-token-elements",
			},
		},
		{
			caseName: "invalid characters replaced with dash success",
			input: inputType{
				elements: []string{
					"invalid/",
					"_characters_",
					"replaced!",
				},
			},
			output: outputType{
				expectedNormalizedClientRequestToken: "invalid---characters--replaced-",
			},
		},
		{
			caseName: "leading dashes trimmed success",
			input: inputType{
				elements: []string{
					"------leading-----",
					"----dashes---",
					"--trimmed-",
				},
			},
			output: outputType{
				expectedNormalizedClientRequestToken: "leading----------dashes------trimmed-",
			},
		},
		{
			caseName: "truncated to 64 characters length success",
			input: inputType{
				elements: []string{
					"023456789",
					"123456789",
					"223456789",
					"323456789",
					"423456789",
					"523456789",
					"623456789",
					"723456789",
				},
			},
			output: outputType{
				expectedNormalizedClientRequestToken: "023456789-123456789-223456789-323456789-423456789-523456789-6234",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualNormalizedClientRequestToken := NewNormalizedClientRequestToken(testCase.input.elements...)

			require.Equal(t, testCase.output.expectedNormalizedClientRequestToken, actualNormalizedClientRequestToken)
		})
	}
}
