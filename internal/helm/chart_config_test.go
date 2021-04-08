// Copyright © 2021 Banzai Cloud
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

package helm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChartConfigIsLessThan(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		caseDescription    string
		expectedIsLessThan bool
		inputConfig        ChartConfig
		inputOtherConfig   ChartConfig
	}{
		{
			caseDescription:    "name is less -> true",
			expectedIsLessThan: true,
			inputConfig: ChartConfig{
				Name:       "name-1",
				Version:    "",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is greater -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-3",
				Version:    "",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is equal, semantic version is less -> true",
			expectedIsLessThan: true,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.0.1",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.0.10",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is equal, semantic version is greater -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.10.0",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.1.0",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is equal, non-semantic receiver version is less -> true",
			expectedIsLessThan: true,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "2.0.0",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is equal, semantic receiver version is greater -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "3",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "2.0.0",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is equal, non-semantic argument version is less -> true",
			expectedIsLessThan: true,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.0.0",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "2",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name is equal, non-semantic argument version is greater -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "3.0.0",
				Repository: "",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "2",
				Repository: "",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name, version are equal, repository is less -> true",
			expectedIsLessThan: true,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-1",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name, version are equal, repository is greater -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-3",
				Values:     nil,
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values:     nil,
			},
		},
		{
			caseDescription:    "name, version, repository are equal, values are less -> true",
			expectedIsLessThan: true,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values: map[string]interface{}{
					"value": 1,
				},
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values: map[string]interface{}{
					"value": 2,
				},
			},
		},
		{
			caseDescription:    "name, version, repository are equal, values are greater -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values: map[string]interface{}{
					"value": 3,
				},
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values: map[string]interface{}{
					"value": 2,
				},
			},
		},
		{
			caseDescription:    "name, version, repository, values are equal -> false",
			expectedIsLessThan: false,
			inputConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values: map[string]interface{}{
					"value": 2,
				},
			},
			inputOtherConfig: ChartConfig{
				Name:       "name-2",
				Version:    "1.2.3",
				Repository: "repository-2",
				Values: map[string]interface{}{
					"value": 2,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			t.Parallel()

			actualIsLessThan := testCase.inputConfig.IsLessThan(testCase.inputOtherConfig)

			require.Equal(t, testCase.expectedIsLessThan, actualIsLessThan)
		})
	}
}
