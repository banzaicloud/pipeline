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

package eksadapter

import (
	"context"
	"fmt"
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon"
	awscommonworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon/awscommonproviders/workflow"
	"github.com/banzaicloud/pipeline/pkg/brn"
)

func TestListNodePools(t *testing.T) {
	type inputType struct {
		cluster   cluster.Cluster
		manager   nodePoolManager
		nodePools map[string]awscommon.ExistingNodePool
	}

	type intermediateDataType struct {
		nodePoolLabels       map[string]map[string]string
		nodePoolDescriptions map[string]*cloudformation.DescribeStacksOutput
	}

	type outputType struct {
		expectedError     error
		expectedNodePools []awscommon.NodePool
	}

	mockMethods := func(
		t *testing.T,
		input inputType,
		intermediateData intermediateDataType,
		mockErrors map[string]error,
	) {
		if mockErrors == nil {
			mockErrors = map[string]error{} // Note: defaulting to nil errors.
		}

		awsSession := &session.Session{}
		cloudFormationAPIClient := &awscommonworkflow.MockcloudFormationAPI{}
		dynamicInterfaceMock := &cluster.MockdynamicInterface{}
		dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}

		schemaGroupVersionResource := schema.GroupVersionResource{
			Group:    "labels.banzaicloud.io",
			Version:  "v1alpha1",
			Resource: "nodepoollabelsets",
		}

		unstructuredList := make([]unstructured.Unstructured, 0, len(input.nodePools))
		for _, nodePool := range input.nodePools {
			labels := intermediateData.nodePoolLabels[nodePool.Name]
			// Note: the client cannot deep copy map[string]string for some reason.
			interfaceLabels := make(map[string]interface{}, len(labels))
			for key, value := range labels {
				interfaceLabels[key] = value
			}

			unstructuredList = append(unstructuredList, unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": fmt.Sprintf("%s/%s", schemaGroupVersionResource.Group, schemaGroupVersionResource.Version),
					"kind":       schemaGroupVersionResource.Resource,
					"metadata": map[string]interface{}{
						"namespace": input.manager.namespace,
						"name":      nodePool.Name,
					},
					"spec": map[string]interface{}{
						"labels": interfaceLabels,
					},
				},
			})
		}

		mocks := make([]string, 0, 6+len(input.nodePools))
		mocks = append(mocks, "DynamicClientFactory.FromSecret")
		mocks = append(mocks, "dynamicInterface.Resource")
		mocks = append(mocks, "dynamicNamespaceableResourceInterface.Namespace")
		mocks = append(mocks, "dynamicNamespaceableResourceInterface.List")
		mocks = append(mocks, "AWSFactory.New")
		mocks = append(mocks, "CloudFormationFactory.New")
		for range input.nodePools {
			mocks = append(mocks, "cloudFormationAPI.DescribeStacks")
		}

		previousMockCounts := make(map[string]int, len(mocks))
		for _, mockID := range mocks {
			switch mockID {
			case "AWSFactory.New":
				mock := input.manager.awsFactory.(*awscommonworkflow.MockAWSFactory).Mock.
					On("New", input.cluster.OrganizationID, input.cluster.SecretID.ResourceID, input.cluster.Location)

				err := mockErrors[mockID]
				if err == nil {
					mock.Return(awsSession, nil).Once()
				} else {
					mock.Return(nil, err).Once()
				}
			case "cloudFormationAPI.DescribeStacks":
				for _, nodePool := range input.nodePools {
					stackIdentifier := nodePool.StackID
					if stackIdentifier == "" {
						stackIdentifier = generateNodePoolStackName(input.cluster.Name, nodePool.Name)
					}

					mock := cloudFormationAPIClient.Mock.
						On("DescribeStacks", &cloudformation.DescribeStacksInput{
							StackName: aws.String(stackIdentifier),
						})

					err := mockErrors[mockID]
					if err == nil {
						mock.Return(intermediateData.nodePoolDescriptions[nodePool.Name], nil).Once()
					} else {
						mock.Return(nil, err).Once()
					}
				}
			case "CloudFormationFactory.New":
				input.manager.cloudFormationFactory.(*awscommonworkflow.MockCloudFormationAPIFactory).Mock.
					On("New", awsSession).
					Return(cloudFormationAPIClient).Once()
			case "DynamicClientFactory.FromSecret":
				mock := input.manager.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory).Mock.
					On("FromSecret", context.Background(), input.cluster.ConfigSecretID.String())

				err := mockErrors[mockID]
				if err == nil {
					mock.Return(dynamicInterfaceMock, nil).Once()
				} else {
					mock.Return(nil, err).Once()
				}
			case "dynamicInterface.Resource":
				dynamicInterfaceMock.Mock.
					On("Resource", schemaGroupVersionResource).
					Return(dynamicResourceInterfaceMock).Once()
			case "dynamicNamespaceableResourceInterface.List":
				mock := dynamicResourceInterfaceMock.Mock.
					On("List", mock.Anything, k8smetav1.ListOptions{})

				err := mockErrors[mockID]
				if err == nil {
					mock.Return(
						&unstructured.UnstructuredList{
							Items: unstructuredList,
						},
						nil,
					).Once()
				} else {
					mock.Return(nil, err).Once()
				}
			case "dynamicNamespaceableResourceInterface.Namespace":
				dynamicResourceInterfaceMock.Mock.
					On("Namespace", input.manager.namespace).
					Return(dynamicResourceInterfaceMock).Once()
			default:
				t.Errorf(
					"unexpected mock call, no mock method is available for this mock ID,"+
						" mock ID: '%s', ordered mock ID occurrences: '%+v'",
					mockID, mocks,
				)
				t.FailNow()
				return
			}

			previousMockCounts[mockID] += 1
		}
	}

	testCases := []struct {
		caseName         string
		input            inputType
		intermediateData intermediateDataType
		mockErrors       map[string]error
		output           outputType
	}{
		{
			caseName: "empty cluster config secret ID error",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
			},
			output: outputType{
				expectedError:     errors.New("cluster is not ready"),
				expectedNodePools: nil,
			},
		},
		{
			caseName: "DynamicClientFactory.FromSecret error",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
			},
			mockErrors: map[string]error{
				"DynamicClientFactory.FromSecret": errors.New("test error: DynamicClientFactory.FromSecret"),
			},
			output: outputType{
				expectedError: errors.New(
					"retrieving node pool label sets failed" +
						": creating dynamic Kubernetes client factory failed" +
						": test error: DynamicClientFactory.FromSecret",
				),
				expectedNodePools: nil,
			},
		},
		{
			caseName: "nodePoolLabelSetManager.GetAll error",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
			},
			mockErrors: map[string]error{
				"dynamicNamespaceableResourceInterface.List": errors.New("test error: nodePoolLabelSetManager.GetAll"),
			},
			output: outputType{
				expectedError: errors.New(
					"retrieving node pool label sets failed" +
						": listing node pool label sets failed" +
						": test error: nodePoolLabelSetManager.GetAll",
				),
				expectedNodePools: nil,
			},
		},
		{
			caseName: "AWSFactory.New error",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
			},
			mockErrors: map[string]error{
				"AWSFactory.New": errors.New("test error: AWSFactory.New"),
			},
			output: outputType{
				expectedError: errors.New(
					"instantiating CloudFormation client failed" +
						": creating aws factory failed" +
						": test error: AWSFactory.New",
				),
				expectedNodePools: nil,
			},
		},
		{
			caseName: "older node pool, missing stack ID and status success",
			input: inputType{
				cluster: cluster.Cluster{
					Name:           "cluster",
					Status:         "UPDATING",
					ConfigSecretID: brn.New(1, "secret", "config-secret-id"),
				},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
				nodePools: map[string]awscommon.ExistingNodePool{
					"older-node-pool-without-stack-id-or-status": {
						Name:          "older-node-pool-without-stack-id-or-status",
						StackID:       "",
						Status:        awscommon.NodePoolStatusEmpty,
						StatusMessage: "",
					},
				},
			},
			mockErrors: map[string]error{
				"cloudFormationAPI.DescribeStacks": errors.New(
					"test error: older node pool, missing stack ID and status success",
				),
			},
			output: outputType{
				expectedError: nil,
				expectedNodePools: []awscommon.NodePool{
					{
						Name:          "older-node-pool-without-stack-id-or-status",
						Status:        awscommon.NodePoolStatusDeleting,
						StatusMessage: "",
					},
				},
			},
		},
		{
			caseName: "node pool creating success",
			input: inputType{
				cluster: cluster.Cluster{
					Name:           "cluster",
					Status:         "UPDATING",
					ConfigSecretID: brn.New(1, "secret", "config-secret-id"),
				},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
				nodePools: map[string]awscommon.ExistingNodePool{
					"creating-pre-stack": {
						Name:          "creating-pre-stack",
						StackID:       "",
						Status:        awscommon.NodePoolStatusCreating,
						StatusMessage: "",
					},
				},
			},
			mockErrors: map[string]error{
				"cloudFormationAPI.DescribeStacks": errors.New("test error: node pool creating pre-stack"),
			},
			output: outputType{
				expectedError: nil,
				expectedNodePools: []awscommon.NodePool{
					{
						Name:          "creating-pre-stack",
						Status:        awscommon.NodePoolStatusCreating,
						StatusMessage: "",
					},
				},
			},
		},
		{
			caseName: "node pool unknown describe failure success",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
				nodePools: map[string]awscommon.ExistingNodePool{
					"unknown-describe-failed": {
						Name:          "unknown-describe-failed",
						StackID:       "unknown-describe-failed/stack-id",
						Status:        awscommon.NodePoolStatusEmpty,
						StatusMessage: "",
					},
				},
			},
			mockErrors: map[string]error{
				"cloudFormationAPI.DescribeStacks": errors.New("test error: node pool unknown describe failure"),
			},
			output: outputType{
				expectedError: nil,
				expectedNodePools: []awscommon.NodePool{
					{
						Name:          "unknown-describe-failed",
						Status:        awscommon.NodePoolStatusUnknown,
						StatusMessage: "retrieving node pool information failed: test error: node pool unknown describe failure",
					},
				},
			},
		},
		{
			caseName: "node pool error stack not found success",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
				nodePools: map[string]awscommon.ExistingNodePool{
					"error-stack-not-found": {
						Name:          "error-stack-not-found",
						StackID:       "error-stack-not-found/stack-id",
						Status:        awscommon.NodePoolStatusEmpty,
						StatusMessage: "",
					},
				},
			},
			intermediateData: intermediateDataType{
				nodePoolDescriptions: map[string]*cloudformation.DescribeStacksOutput{
					"error-stack-not-found": {},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedNodePools: []awscommon.NodePool{
					{
						Name:          "error-stack-not-found",
						Status:        awscommon.NodePoolStatusUnknown,
						StatusMessage: "retrieving node pool information failed: node pool not found",
					},
				},
			},
		},
		{
			caseName: "node pool error multiple stacks found success",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
				nodePools: map[string]awscommon.ExistingNodePool{
					"error-multiple-stacks-found": {
						Name:          "error-multiple-stacks-found",
						StackID:       "error-multiple-stacks-found/stack-id",
						Status:        awscommon.NodePoolStatusEmpty,
						StatusMessage: "",
					},
				},
			},
			intermediateData: intermediateDataType{
				nodePoolDescriptions: map[string]*cloudformation.DescribeStacksOutput{
					"error-multiple-stacks-found": {
						Stacks: []*cloudformation.Stack{
							{},
							{},
						},
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedNodePools: []awscommon.NodePool{
					{
						Name:          "error-multiple-stacks-found",
						Status:        awscommon.NodePoolStatusUnknown,
						StatusMessage: "retrieving node pool information failed: multiple node pools found",
					},
				},
			},
		},
		{
			caseName: "multiple node pools ready, updating success",
			input: inputType{
				cluster: cluster.Cluster{ConfigSecretID: brn.New(1, "secret", "config-secret-id")},
				manager: nodePoolManager{
					awsFactory:            &awscommonworkflow.MockAWSFactory{},
					cloudFormationFactory: &awscommonworkflow.MockCloudFormationAPIFactory{},
					dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				},
				nodePools: map[string]awscommon.ExistingNodePool{
					"ready": {
						Name:          "ready",
						StackID:       "ready/stack-id",
						Status:        awscommon.NodePoolStatusEmpty,
						StatusMessage: "",
					},
					"updating": {
						Name:          "updating",
						StackID:       "updating/stack-id",
						Status:        awscommon.NodePoolStatusEmpty,
						StatusMessage: "",
					},
				},
			},
			intermediateData: intermediateDataType{
				nodePoolLabels: map[string]map[string]string{
					"ready": {
						"label-key-1": "label-value-1",
						"label-key-2": "label-value-2",
					},
					"updating": {
						"label-key-3": "label-value-3",
						"label-key-4": "label-value-4",
					},
				},
				nodePoolDescriptions: map[string]*cloudformation.DescribeStacksOutput{
					"ready": {
						Stacks: []*cloudformation.Stack{
							{
								Parameters: []*cloudformation.Parameter{
									{
										ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
										ParameterValue: aws.String("true"),
									},
									{
										ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
										ParameterValue: aws.String("2"),
									},
									{
										ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
										ParameterValue: aws.String("1"),
									},
									{
										ParameterKey:   aws.String("NodeAutoScalingInitSize"),
										ParameterValue: aws.String("1"),
									},
									{
										ParameterKey:   aws.String("NodeImageId"),
										ParameterValue: aws.String("ami-0123456789"),
									},
									{
										ParameterKey:   aws.String("NodeInstanceType"),
										ParameterValue: aws.String("t2.small"),
									},
									{
										ParameterKey:   aws.String("NodeSpotPrice"),
										ParameterValue: aws.String("0.02"),
									},
									{
										ParameterKey:   aws.String("NodeVolumeSize"),
										ParameterValue: aws.String("20"),
									},
									{
										ParameterKey:   aws.String("Subnets"),
										ParameterValue: aws.String("subnet-0123456789"),
									},
								},
								StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
							},
						},
					},
					"updating": {
						Stacks: []*cloudformation.Stack{
							{
								Parameters: []*cloudformation.Parameter{
									{
										ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
										ParameterValue: aws.String("false"),
									},
									{
										ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
										ParameterValue: aws.String("0"),
									},
									{
										ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
										ParameterValue: aws.String("0"),
									},
									{
										ParameterKey:   aws.String("NodeAutoScalingInitSize"),
										ParameterValue: aws.String("5"),
									},
									{
										ParameterKey:   aws.String("NodeImageId"),
										ParameterValue: aws.String("ami-1234567890"),
									},
									{
										ParameterKey:   aws.String("NodeInstanceType"),
										ParameterValue: aws.String("t2.medium"),
									},
									{
										ParameterKey:   aws.String("NodeSpotPrice"),
										ParameterValue: aws.String("0.01"),
									},
									{
										ParameterKey:   aws.String("NodeVolumeSize"),
										ParameterValue: aws.String("25"),
									},
									{
										ParameterKey:   aws.String("Subnets"),
										ParameterValue: aws.String("subnet-1234567890"),
									},
								},
								StackStatus: aws.String(cloudformation.StackStatusUpdateInProgress),
							},
						},
					},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedNodePools: []awscommon.NodePool{
					{
						Name: "ready",
						Labels: map[string]string{
							"label-key-1": "label-value-1",
							"label-key-2": "label-value-2",
						},
						Size: 1,
						Autoscaling: awscommon.Autoscaling{
							Enabled: true,
							MinSize: 1,
							MaxSize: 2,
						},
						VolumeSize:    20,
						InstanceType:  "t2.small",
						Image:         "ami-0123456789",
						SpotPrice:     "0.02",
						SubnetID:      "subnet-0123456789",
						Status:        awscommon.NodePoolStatusReady,
						StatusMessage: "",
					},
					{
						Name: "updating",
						Labels: map[string]string{
							"label-key-3": "label-value-3",
							"label-key-4": "label-value-4",
						},
						Size: 5,
						Autoscaling: awscommon.Autoscaling{
							Enabled: false,
							MinSize: 0,
							MaxSize: 0,
						},
						VolumeSize:    25,
						InstanceType:  "t2.medium",
						Image:         "ami-1234567890",
						SpotPrice:     "0.01",
						SubnetID:      "subnet-1234567890",
						Status:        awscommon.NodePoolStatusUpdating,
						StatusMessage: "",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			mockMethods(t, testCase.input, testCase.intermediateData, testCase.mockErrors)

			actualNodePools, actualError := testCase.input.manager.ListNodePools(
				context.Background(),
				testCase.input.cluster,
				testCase.input.nodePools,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedNodePools, actualNodePools)
		})
	}
}
