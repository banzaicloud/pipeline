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
	"fmt"

	"emperror.dev/errors"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/arn"
)

const (
	// awsAuthConfigMapTemplate is a helper template for generating AWS auth
	// config maps.
	awsAuthConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
%s
  mapUsers: |
%s`

	// mapRolesTemplate is a helper template for generating the map roles
	// section of the AWS auth config map.
	mapRolesTemplate = `    - rolearn: %s
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes
`

	// mapUsersTemplate is a helper template for generating the map users
	// section of the AWS auth config map.
	mapUsersTemplate = `    - userarn: %s
      username: %s
      groups:
      - system:masters
`
)

type mergeFunc func(key, defaultData, inputData string) ([]byte, error)

// MergeAuthConfigMaps merges tworaw AWS auth config maps into one.
func MergeAuthConfigMaps(defaultConfigMap, inputConfigMap string) (*v1.ConfigMap, error) {
	var defaultAWSConfigMap v1.ConfigMap
	_, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(defaultConfigMap), nil, &defaultAWSConfigMap)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode default config map")
	}

	var inputAWSConfigMap v1.ConfigMap
	_, _, err = scheme.Codecs.UniversalDeserializer().Decode([]byte(inputConfigMap), nil, &inputAWSConfigMap)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode input config map")
	}

	mergedConfigMap := defaultAWSConfigMap
	if mergedConfigMap.Data == nil {
		mergedConfigMap.Data = make(map[string]string, len(inputAWSConfigMap.Data))
	}

	functions := map[string]mergeFunc{
		"mapUsers":    mergeDefaultMapWithInput,
		"mapRoles":    mergeDefaultMapWithInput,
		"mapAccounts": mergeDefaultStrArrayWithInput,
	}

	for key, merge := range functions {
		mergedValue, err := merge(key, defaultAWSConfigMap.Data[key], inputAWSConfigMap.Data[key])
		if err != nil {
			return nil, errors.WrapIf(err, "failed to merge "+key+" default value and input")
		}

		mergedConfigMap.Data[key] = string(mergedValue)
	}

	return &mergedConfigMap, nil
}

func mergeDefaultStrArrayWithInput(key, defaultData, inputData string) ([]byte, error) {
	var defaultValue []string
	err := yaml.Unmarshal([]byte(defaultData), &defaultValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+defaultData)
	}

	var inputValue []string
	err = yaml.Unmarshal([]byte(inputData), &inputValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+inputData)
	}

	var result []byte
	value := append(defaultValue, inputValue...)
	if value != nil {
		result, err = yaml.Marshal(value)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to marshal "+key+" to yaml")
		}
	}

	return result, nil
}

func mergeDefaultMapWithInput(key, defaultData, inputData string) ([]byte, error) {
	var defaultValue []map[string]interface{}
	err := yaml.Unmarshal([]byte(defaultData), &defaultValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+defaultData)
	}

	var inputValue []map[string]interface{}
	err = yaml.Unmarshal([]byte(inputData), &inputValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+inputData)
	}

	var result []byte
	value := append(defaultValue, inputValue...)
	if value != nil {
		result, err = yaml.Marshal(value)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to marshal "+key+" to yaml")
		}
	}

	return result, nil
}

// NewDefaultAWSAuthConfigMap constructs an AWS auth config map from the
// specified cluster user and node instance role.
func NewDefaultAWSAuthConfigMap(clusterUserARN, nodeInstanceRoleARN string) (defaultAWSAuthConfigMap string) {
	mapRoles := fmt.Sprintf(mapRolesTemplate, nodeInstanceRoleARN)

	// The aws-iam-authenticator doesn't handle path currently in role mappings:
	// https://github.com/kubernetes-sigs/aws-iam-authenticator/issues/268
	// Once this bug gets fixed our code won't work, so we are making it future
	// compatible by adding the role id with and without path to the mapping.
	if arn.ResourcePathOrParent(nodeInstanceRoleARN) != arn.ARNPathSeparator {
		pathlessRoleARN := arn.NewARN(
			arn.Partition(nodeInstanceRoleARN),
			arn.Service(nodeInstanceRoleARN),
			arn.Region(nodeInstanceRoleARN),
			arn.AccountID(nodeInstanceRoleARN),
			arn.ResourceType(nodeInstanceRoleARN),
			arn.ARNPathSeparator,
			arn.ResourceName(nodeInstanceRoleARN),
			arn.ResourceQualifier(nodeInstanceRoleARN),
		)
		mapRoles += fmt.Sprintf(mapRolesTemplate, pathlessRoleARN)
	}

	mapUsers := fmt.Sprintf(mapUsersTemplate, clusterUserARN, arn.ResourceName(clusterUserARN))

	return fmt.Sprintf(awsAuthConfigMapTemplate, mapRoles, mapUsers)
}
