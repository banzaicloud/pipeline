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

package driver

import (
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster"
)

func TestValidateEncryptionConfiguration(t *testing.T) {
	type inputType struct {
		clusterLocation  string
		encryptionConfig []ekscluster.EncryptionConfig
	}

	type outputType struct {
		expectedError error
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "nil encryptionConfig",
			input: inputType{
				clusterLocation:  "",
				encryptionConfig: nil,
			},
			output: outputType{
				expectedError: nil,
			},
		},
		{
			caseName: "empty encryptionConfig",
			input: inputType{
				clusterLocation:  "",
				encryptionConfig: []ekscluster.EncryptionConfig{},
			},
			output: outputType{
				expectedError: nil,
			},
		},
		{
			caseName: "multiple configurations encryptionConfig error",
			input: inputType{
				clusterLocation: "",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{},
					{},
				},
			},
			output: outputType{
				expectedError: errors.New("invalid encryption configuration item count"),
			},
		},
		{
			caseName: "empty encryptionConfig[0].Provider.KeyARN error",
			input: inputType{
				clusterLocation: "",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "",
						},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid empty keyARN value"),
			},
		},
		{
			caseName: "invalid encryptionConfig[0].Provider.KeyARN error",
			input: inputType{
				clusterLocation: "test-cluster-location",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "random",
						},
						Resources: []string{"random"},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid non-KMS ARN or non-ARN value specified"),
			},
		},
		{
			caseName: "nil encryptionConfig.Resources error",
			input: inputType{
				clusterLocation: "",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "arn:aws:kms:test-cluster-location:test-account-id:key/13b3cb50-53df-598g-c117-45d75g5c871f",
						},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid nil resources value"),
			},
		},
		{
			caseName: "empty encryptionConfig.Resources error",
			input: inputType{
				clusterLocation: "",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "arn:aws:kms:test-cluster-location:test-account-id:key/13b3cb50-53df-598g-c117-45d75g5c871f",
						},
						Resources: []string{},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid encryption configuration resource count"),
			},
		},
		{
			caseName: "invalid encryptionConfig[0].Resources[0] error",
			input: inputType{
				clusterLocation: "test-cluster-location",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "arn:aws:kms:test-cluster-location:test-account-id:key/13b3cb50-53df-598g-c117-45d75g5c871f",
						},
						Resources: []string{"random"},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid encryption config resource, only allowed value is 'secrets'"),
			},
		},
		{
			caseName: "empty clusterLocation error",
			input: inputType{
				clusterLocation: "",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "arn:aws:kms:test-cluster-location:test-account-id:key/13b3cb50-53df-598g-c117-45d75g5c871f",
						},
						Resources: []string{"secrets"},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid empty cluster location"),
			},
		},
		{
			caseName: "mismatching key and cluster location error",
			input: inputType{
				clusterLocation: "test-cluster-location-2",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "arn:aws:kms:test-cluster-location:test-account-id:key/13b3cb50-53df-598g-c117-45d75g5c871f",
						},
						Resources: []string{"secrets"},
					},
				},
			},
			output: outputType{
				expectedError: errors.NewWithDetails("invalid key, cluster and key locations mismatch"),
			},
		},
		{
			caseName: "valid encryption",
			input: inputType{
				clusterLocation: "test-cluster-location",
				encryptionConfig: []ekscluster.EncryptionConfig{
					{
						Provider: ekscluster.Provider{
							KeyARN: "arn:aws:kms:test-cluster-location:test-account-id:key/13b3cb50-53df-598g-c117-45d75g5c871f",
						},
						Resources: []string{"secrets"},
					},
				},
			},
			output: outputType{
				expectedError: nil,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			actualError := validateEncryptionConfiguration(
				testCase.input.encryptionConfig,
				testCase.input.clusterLocation,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
		})
	}
}
