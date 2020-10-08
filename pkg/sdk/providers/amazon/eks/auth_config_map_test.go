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

package eks

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergeAuthConfigMaps(t *testing.T) {
	type inputType struct {
		defaultConfigMap string
		inputConfigMap   string
	}

	type outputType struct {
		expectedConfigMap *corev1.ConfigMap
		expectedError     error
	}

	testCases := []struct {
		caseName string
		input    inputType
		output   outputType
	}{
		{
			caseName: "empty values success",
			input: inputType{
				defaultConfigMap: "",
				inputConfigMap:   "",
			},
			output: outputType{
				expectedConfigMap: &corev1.ConfigMap{
					Data: map[string]string{
						"mapAccounts": "",
						"mapRoles":    "",
						"mapUsers":    "",
					},
				},
				expectedError: nil,
			},
		},
		{
			caseName: "invalid default config map error",
			input: inputType{
				defaultConfigMap: "- this is not a valid config map",
				inputConfigMap:   "",
			},
			output: outputType{
				expectedConfigMap: nil,
				expectedError:     errors.New("failed to decode default config map: couldn't get version/kind; json parse error: json: cannot unmarshal array into Go value of type struct { APIVersion string \"json:\\\"apiVersion,omitempty\\\"\"; Kind string \"json:\\\"kind,omitempty\\\"\" }"),
			},
		},
		{
			caseName: "invalid input config map error",
			input: inputType{
				defaultConfigMap: "",
				inputConfigMap:   "- this is not a valid config map",
			},
			output: outputType{
				expectedConfigMap: nil,
				expectedError:     errors.New("failed to decode input config map: couldn't get version/kind; json parse error: json: cannot unmarshal array into Go value of type struct { APIVersion string \"json:\\\"apiVersion,omitempty\\\"\"; Kind string \"json:\\\"kind,omitempty\\\"\" }"),
			},
		},
		{
			caseName: "merge error",
			input: inputType{
				defaultConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    associative: true
`,
				inputConfigMap: "",
			},
			output: outputType{
				expectedConfigMap: nil,
				expectedError:     errors.New("failed to merge mapRoles default value and input: failed to unmarshal mapRoles from: associative: true\n: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!map into []map[string]interface {}"),
			},
		},
		{
			caseName: "ARNs without path success",
			input: inputType{
				defaultConfigMap: NewDefaultAWSAuthConfigMap(
					"arn:partition:service:region:accountID:type/name",
					"arn:partition2:service2:region2:accountID2:type2/name2",
				),
				inputConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    - groups:
      - system:bootstrappers
      - system:nodes
      rolearn: arn:aws:iam::555555555555:role/devel-nodes-NodeInstanceRole-74RF4UBDUKL6
      username: system:node:{{EC2PrivateDNSName}}
    - groups:
      - system:bootstrappers
      - system:nodes
      rolearn: arn:aws:iam::555555555555:role/devel-nodes-NodeInstanceRole-74RF4UBDUKL6-2
      username: system:node:{{EC2PrivateDNSName}}

  mapUsers: |
    - groups:
      - system:masters
      userarn: arn:aws:iam::555555555555:user/admin
      username: admin
    - groups:
      - system:masters
      userarn: arn:aws:iam::111122223333:user/ops-user
      username: ops-user

  mapAccounts: |
    - account1
    - otheraccount
`,
			},
			output: outputType{
				expectedConfigMap: &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-auth",
						Namespace: "kube-system",
					},
					Data: map[string]string{
						"mapAccounts": `- account1
- otheraccount
`,
						"mapRoles": `- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:partition2:service2:region2:accountID2:type2/name2
  username: system:node:{{EC2PrivateDNSName}}
- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:aws:iam::555555555555:role/devel-nodes-NodeInstanceRole-74RF4UBDUKL6
  username: system:node:{{EC2PrivateDNSName}}
- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:aws:iam::555555555555:role/devel-nodes-NodeInstanceRole-74RF4UBDUKL6-2
  username: system:node:{{EC2PrivateDNSName}}
`,
						"mapUsers": `- groups:
  - system:masters
  userarn: arn:partition:service:region:accountID:type/name
  username: name
- groups:
  - system:masters
  userarn: arn:aws:iam::555555555555:user/admin
  username: admin
- groups:
  - system:masters
  userarn: arn:aws:iam::111122223333:user/ops-user
  username: ops-user
`,
					},
				},
				expectedError: nil,
			},
		},
		{
			caseName: "ARNs with path success",
			input: inputType{
				defaultConfigMap: NewDefaultAWSAuthConfigMap(
					"arn:partition:service:region:accountID:type/additional/path/elements/name",
					"arn:partition2:service2:region2:accountID2:type2/additional/path/elements/name2",
				),
				inputConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    - groups:
      - system:bootstrappers
      - system:nodes
      rolearn: arn:aws:iam::555555555555:role/additional/path/elements/devel-nodes-NodeInstanceRole-74RF4UBDUKL6
      username: system:node:{{EC2PrivateDNSName}}
    - groups:
      - system:bootstrappers
      - system:nodes
      rolearn: arn:aws:iam::555555555555:role/additional/path/elements/devel-nodes-NodeInstanceRole-74RF4UBDUKL6-2
      username: system:node:{{EC2PrivateDNSName}}

  mapUsers: |
    - groups:
      - system:masters
      userarn: arn:aws:iam::555555555555:user/additional/path/elements/admin
      username: admin
    - groups:
      - system:masters
      userarn: arn:aws:iam::111122223333:user/additional/path/elements/ops-user
      username: ops-user

  mapAccounts: |
    - account1
    - otheraccount
`,
			},
			output: outputType{
				expectedConfigMap: &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-auth",
						Namespace: "kube-system",
					},
					Data: map[string]string{
						"mapAccounts": `- account1
- otheraccount
`,
						"mapRoles": `- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:partition2:service2:region2:accountID2:type2/additional/path/elements/name2
  username: system:node:{{EC2PrivateDNSName}}
- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:partition2:service2:region2:accountID2:type2/name2
  username: system:node:{{EC2PrivateDNSName}}
- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:aws:iam::555555555555:role/additional/path/elements/devel-nodes-NodeInstanceRole-74RF4UBDUKL6
  username: system:node:{{EC2PrivateDNSName}}
- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:aws:iam::555555555555:role/additional/path/elements/devel-nodes-NodeInstanceRole-74RF4UBDUKL6-2
  username: system:node:{{EC2PrivateDNSName}}
`,
						"mapUsers": `- groups:
  - system:masters
  userarn: arn:partition:service:region:accountID:type/additional/path/elements/name
  username: name
- groups:
  - system:masters
  userarn: arn:aws:iam::555555555555:user/additional/path/elements/admin
  username: admin
- groups:
  - system:masters
  userarn: arn:aws:iam::111122223333:user/additional/path/elements/ops-user
  username: ops-user
`,
					},
				},
				expectedError: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualConfigMap, actualError := MergeAuthConfigMaps(
				testCase.input.defaultConfigMap,
				testCase.input.inputConfigMap,
			)

			if testCase.output.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedConfigMap, actualConfigMap)
		})
	}
}

func TestNewDefaultAWSAuthConfigMap(t *testing.T) {
	type inputType struct {
		clusterUserARN      string
		nodeInstanceRoleARN string
	}

	testCases := []struct {
		caseName                        string
		expectedDefaultAWSAuthConfigMap string
		input                           inputType
	}{
		{
			caseName: "empty values success",
			expectedDefaultAWSAuthConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    - rolearn: ` + `
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes
    - rolearn: arn:::::/
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes

  mapUsers: |
    - userarn: ` + `
      username: ` + `
      groups:
      - system:masters
`, // Note: the ` + ` breaks are there to work around auto-formatting removing the trailing space.
			input: inputType{
				clusterUserARN:      "",
				nodeInstanceRoleARN: "",
			},
		},
		{
			caseName: "ARNs without path success",
			expectedDefaultAWSAuthConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    - rolearn: arn:partition2:service2:region2:accountID2:type2/name2
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes

  mapUsers: |
    - userarn: arn:partition:service:region:accountID:type/name
      username: name
      groups:
      - system:masters
`,
			input: inputType{
				clusterUserARN:      "arn:partition:service:region:accountID:type/name",
				nodeInstanceRoleARN: "arn:partition2:service2:region2:accountID2:type2/name2",
			},
		},
		{
			caseName: "ARNs with path success",
			expectedDefaultAWSAuthConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
    - rolearn: arn:partition2:service2:region2:accountID2:type2/additional/path/elements/name2
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes
    - rolearn: arn:partition2:service2:region2:accountID2:type2/name2
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes

  mapUsers: |
    - userarn: arn:partition:service:region:accountID:type/additional/path/elements/name
      username: name
      groups:
      - system:masters
`,
			input: inputType{
				clusterUserARN:      "arn:partition:service:region:accountID:type/additional/path/elements/name",
				nodeInstanceRoleARN: "arn:partition2:service2:region2:accountID2:type2/additional/path/elements/name2",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualDefaultAWSAuthConfigMap := NewDefaultAWSAuthConfigMap(
				testCase.input.clusterUserARN,
				testCase.input.nodeInstanceRoleARN,
			)

			require.Equal(t, testCase.expectedDefaultAWSAuthConfigMap, actualDefaultAWSAuthConfigMap)
		})
	}
}
