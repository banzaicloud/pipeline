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

package cmd

import (
	"sort"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/helm"
)

func TestIsConfigEnabled(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		caseDescription         string
		expectedIsConfigEnabled bool
		inputConfig             interface{}
	}{
		{
			caseDescription:         "nil config -> false",
			expectedIsConfigEnabled: false,
			inputConfig:             nil,
		},
		{
			caseDescription:         "not associative type config -> true",
			expectedIsConfigEnabled: true,
			inputConfig:             []string{},
		},
		{
			caseDescription:         "config cannot be disabled -> true",
			expectedIsConfigEnabled: true,
			inputConfig:             struct{ Value int }{},
		},
		{
			caseDescription:         "config disabled -> false",
			expectedIsConfigEnabled: false,
			inputConfig: struct {
				Enabled bool
				Chart   string
				Version string
				Values  map[string]interface{}
			}{
				Enabled: false,
			},
		},
		{
			caseDescription:         "config enabled -> true",
			expectedIsConfigEnabled: true,
			inputConfig: struct {
				Enabled bool
				Chart   string
				Version string
				Values  map[string]interface{}
			}{
				Enabled: true,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			t.Parallel()

			actualIsConfigEnabled := isConfigEnabled(testCase.inputConfig)

			require.Equal(t, testCase.expectedIsConfigEnabled, actualIsConfigEnabled)
		})
	}
}

func TestParseClusterChartConfigsRecursively(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		caseDescription             string
		expectedClusterChartConfigs []helm.ChartConfig
		inputConfig                 interface{}
		inputDecoder                *mapstructure.Decoder
		inputDecoderConfig          *mapstructure.DecoderConfig
	}{
		{
			caseDescription:             "nil config -> nil chart configs",
			expectedClusterChartConfigs: nil,
			inputConfig:                 nil,
			inputDecoder:                nil,
			inputDecoderConfig:          nil,
		},
		{
			caseDescription:             "basic type config -> nil chart configs",
			expectedClusterChartConfigs: nil,
			inputConfig:                 "basic-config",
			inputDecoder:                nil,
			inputDecoderConfig:          nil,
		},
		{
			caseDescription:             "disabled config -> nil chart configs",
			expectedClusterChartConfigs: nil,
			inputConfig: struct {
				Enabled bool
				Chart   string
				Version string
				Values  map[string]interface{}
			}{
				Enabled: false,
				Chart:   "repository/chart",
				Version: "1.2.3",
				Values: map[string]interface{}{
					"Value": 5,
				},
			},
			inputDecoder:       nil,
			inputDecoderConfig: nil,
		},
		{
			caseDescription: "top level chart -> single chart config",
			expectedClusterChartConfigs: []helm.ChartConfig{
				{
					Name:       "chart",
					Version:    "1.2.3",
					Repository: "repository",
					Values: map[string]interface{}{
						"Value": 5,
					},
				},
			},
			inputConfig: struct {
				Chart   string
				Version string
				Values  map[string]interface{}
			}{
				Chart:   "repository/chart",
				Version: "1.2.3",
				Values: map[string]interface{}{
					"Value": 5,
				},
			},
			inputDecoder:       nil,
			inputDecoderConfig: nil,
		},
		{
			caseDescription: "deep chart configs -> chart configs",
			expectedClusterChartConfigs: []helm.ChartConfig{
				{
					Name:       "x-chart",
					Version:    "0.1.2",
					Repository: "",
					Values:     nil,
				},
				{
					Name:       "y-chart",
					Version:    "1.2.3",
					Repository: "repository",
					Values: map[string]interface{}{
						"Value": 5,
					},
				},
				{
					Name:       "z-chart",
					Version:    "2.3.4",
					Repository: "repository-2",
					Values: map[string]interface{}{
						"Value": 10,
					},
				},
			},
			inputConfig: map[string]interface{}{
				"Components": []interface{}{
					[]interface{}{ // Note: enabled charts.
						map[string]interface{}{
							"X": map[string]interface{}{
								"Chart":   "x-chart",
								"Version": "0.1.2",
							},
						},
						map[string]interface{}{
							"Y": map[string]interface{}{
								"Chart":   "repository/y-chart",
								"Version": "1.2.3",
								"Values": map[string]interface{}{
									"Value": 5,
								},
							},
						},
						map[string]interface{}{
							"Z": map[string]interface{}{
								"Chart":   "repository-2/z-chart",
								"Version": "2.3.4",
								"Values": map[string]interface{}{
									"Value": 10,
								},
							},
						},
					},
					map[string]interface{}{
						"Enabled": false,
						"Subcharts": []interface{}{
							map[string]interface{}{
								"A": map[string]interface{}{
									"Chart":   "a-chart",
									"Version": "3.4.5",
								},
							},
							map[string]interface{}{
								"B": map[string]interface{}{
									"Chart":   "repository/y-chart",
									"Version": "1.2.3",
									"Values": map[string]interface{}{
										"Value": 5,
									},
								},
							},
							map[string]interface{}{
								"Z": map[string]interface{}{
									"Chart":   "repository/z-chart",
									"Version": "2.3.4",
									"Values": map[string]interface{}{
										"Value": 10,
									},
								},
							},
						},
					},
				},
			},
			inputDecoder:       nil,
			inputDecoderConfig: nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			t.Parallel()

			actualClusterChartConfigs := parseClusterChartConfigsRecursively(
				testCase.inputDecoder,
				testCase.inputDecoderConfig,
				testCase.inputConfig,
			)

			sort.Slice(testCase.expectedClusterChartConfigs, func(firstIndex, secondIndex int) (issLessThan bool) {
				return testCase.expectedClusterChartConfigs[firstIndex].IsLessThan(
					testCase.expectedClusterChartConfigs[secondIndex],
				)
			})
			sort.Slice(actualClusterChartConfigs, func(firstIndex, secondIndex int) (issLessThan bool) {
				return actualClusterChartConfigs[firstIndex].IsLessThan(actualClusterChartConfigs[secondIndex])
			})

			require.Equal(t, testCase.expectedClusterChartConfigs, actualClusterChartConfigs)
		})
	}
}
