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
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/client"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/pkg/brn"
)

// newFakeUnstructuredObjectWithSpec creates an unstructured Kubernetes resource object
// for test fake purposes with the specified necessary and optional information.
func newFakeUnstructuredObjectWithSpec(apiVersion, kind, name, optionalNamespace string, optionalSpec map[string]interface{}) (object unstructured.Unstructured) {
	if optionalNamespace == "" {
		optionalNamespace = "default"
	}

	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": optionalNamespace,
				"name":      name,
			},
			"spec": optionalSpec,
		},
	}
}

func TestListNodePools(t *testing.T) {
	exampleAWSClient := &session.Session{}
	exampleClusterID := uint(0)
	exampleContext := context.Background()
	var exampleEnterprise bool
	exampleLabels := map[string]string{
		"label-key": "value",
	}
	exampleNamespace := "namespace"
	exampleNodePoolNames := []string{
		"node-pool-name-1",
		"node-pool-name-2",
	}
	exampleOrganizationID := uint(1)
	exampleSchemaGroupVersionResource := schema.GroupVersionResource{
		Group:    "labels.banzaicloud.io",
		Version:  "v1alpha1",
		Resource: "nodepoollabelsets",
	}
	exampleStackParameters := map[string]interface{}{
		"ClusterAutoscalerEnabled":    true,
		"NodeAutoScalingGroupMaxSize": 1,
		"NodeAutoScalingGroupMinSize": 3,
		"NodeAutoScalingInitSize":     2,
		"NodeVolumeSize":              50,
		"NodeInstanceType":            "node pool instance type",
		"NodeImageId":                 "node pool image ID",
		"NodeSpotPrice":               "node pool spot price",
	}
	var exampleWorkflowClient client.Client
	//
	exampleCluster := cluster.Cluster{
		ID:             exampleClusterID,
		UID:            "cluster UID",
		Name:           "cluster name",
		OrganizationID: exampleOrganizationID,
		Status:         "cluster status",
		StatusMessage:  "cluster status message",
		Cloud:          "cluster cloud",
		Distribution:   "cluster distribution",
		Location:       "cluster location",
		SecretID: brn.ResourceName{
			Scheme:         "cluster secret ID scheme",
			OrganizationID: exampleOrganizationID,
			ResourceType:   "cluster secret ID resource type",
			ResourceID:     "cluster secret ID resource ID",
		},
		ConfigSecretID: brn.ResourceName{
			Scheme:         "cluster config secret ID scheme",
			OrganizationID: exampleOrganizationID,
			ResourceType:   "cluster config secret ID resource type",
			ResourceID:     "cluster config secret ID resource ID",
		},
		Tags: map[string]string{
			"cluster-tag": "cluster tag value",
		},
	}
	exampleDescribeStacksOutput := &cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				Parameters: []*cloudformation.Parameter{},
			},
		},
	}
	for parameterKey, parameterValue := range exampleStackParameters {
		parameterKeyString := parameterKey // Note: Parameters requires a string pointer which shouldn't be the iterator.
		parameterValueString := fmt.Sprintf("%+v", parameterValue)
		exampleDescribeStacksOutput.Stacks[0].Parameters = append(exampleDescribeStacksOutput.Stacks[0].Parameters, &cloudformation.Parameter{
			ParameterKey:   &parameterKeyString,
			ParameterValue: &parameterValueString,
		})
	}
	exampleLabelsWithInterface := make(map[string]interface{}, len(exampleLabels))
	for key, value := range exampleLabels {
		exampleLabelsWithInterface[key] = value
	}
	exampleNodePool := eks.NodePool{
		Name:   "replace me",
		Labels: exampleLabels,
		Size:   exampleStackParameters["NodeAutoScalingInitSize"].(int),
		Autoscaling: eks.Autoscaling{
			Enabled: exampleStackParameters["ClusterAutoscalerEnabled"].(bool),
			MinSize: exampleStackParameters["NodeAutoScalingGroupMinSize"].(int),
			MaxSize: exampleStackParameters["NodeAutoScalingGroupMaxSize"].(int),
		},
		VolumeSize:   exampleStackParameters["NodeVolumeSize"].(int),
		InstanceType: exampleStackParameters["NodeInstanceType"].(string),
		Image:        exampleStackParameters["NodeImageId"].(string),
		SpotPrice:    exampleStackParameters["NodeSpotPrice"].(string),
	}
	//
	exampleUnstructuredList := make([]unstructured.Unstructured, len(exampleNodePoolNames))
	for nodePoolNameIndex, nodePoolName := range exampleNodePoolNames {
		exampleUnstructuredList[nodePoolNameIndex] = newFakeUnstructuredObjectWithSpec(
			fmt.Sprintf("%s/%s", exampleSchemaGroupVersionResource.Group, exampleSchemaGroupVersionResource.Version),
			exampleSchemaGroupVersionResource.Resource,
			nodePoolName,
			exampleNamespace,
			map[string]interface{}{
				"labels": exampleLabelsWithInterface, // Note: the client cannot deep copy map[string]string for some reason.
			},
		)
	}
	exampleNodePools := make([]eks.NodePool, len(exampleNodePoolNames))
	for nodePoolIndex, nodePoolName := range exampleNodePoolNames {
		exampleNodePools[nodePoolIndex] = exampleNodePool
		exampleNodePools[nodePoolIndex].Name = nodePoolName
	}
	//
	exampleClusterClientObjects := make([]runtime.Object, len(exampleUnstructuredList))
	for objectIndex, object := range exampleUnstructuredList {
		object := object
		exampleClusterClientObjects[objectIndex] = &object
	}

	type constructionArgumentType struct {
		awsFactory            workflow.AWSFactory
		cloudFormationFactory workflow.CloudFormationAPIFactory
		dynamicClientFactory  cluster.DynamicKubeClientFactory
		enterprise            bool
		namespace             string
		workflowClient        client.Client
	}
	type functionCallArgumentType struct {
		ctx           context.Context
		cluster       cluster.Cluster
		nodePoolNames []string
	}
	testCases := []struct {
		caseName              string
		constructionArguments constructionArgumentType
		expectedNodePools     []eks.NodePool
		expectedNotNilError   bool
		functionCallArguments functionCallArgumentType
		setupMocks            func(constructionArgumentType, functionCallArgumentType)
	}{
		{
			caseName: "DynamicKubeClientFactoryFromSecretError",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamic.Interface(nil), errors.NewWithDetails("DynamicKubeClientFactoryFromSecretError"))
			},
		},
		{
			caseName: "NodePoolLabelSetManagerGetAllError",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}
				dynamicResourceInterfaceMock.On("Namespace", exampleNamespace).Return(dynamicResourceInterfaceMock)
				dynamicResourceInterfaceMock.On("List", mock.Anything, k8smetav1.ListOptions{}).Return(nil, errors.NewWithDetails("NodePoolLabelSetManagerGetAllError"))

				dynamicInterfaceMock := &cluster.MockdynamicInterface{}
				dynamicInterfaceMock.On("Resource", exampleSchemaGroupVersionResource).Return(dynamicResourceInterfaceMock)

				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamicInterfaceMock, (error)(nil))
			},
		},
		{
			caseName: "AWSClientFactoryNewError",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}
				dynamicResourceInterfaceMock.On("Namespace", exampleNamespace).Return(dynamicResourceInterfaceMock)
				dynamicResourceInterfaceMock.On("List", mock.Anything, k8smetav1.ListOptions{}).Return(&unstructured.UnstructuredList{Items: exampleUnstructuredList}, (error)(nil))

				dynamicInterfaceMock := &cluster.MockdynamicInterface{}
				dynamicInterfaceMock.On("Resource", exampleSchemaGroupVersionResource).Return(dynamicResourceInterfaceMock)

				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamicInterfaceMock, (error)(nil))

				awsFactoryMock := constructionArguments.awsFactory.(*workflow.MockAWSFactory)
				awsFactoryMock.On("New", functionCallArguments.cluster.OrganizationID, functionCallArguments.cluster.SecretID.ResourceID, functionCallArguments.cluster.Location).Return((*session.Session)(nil), errors.NewWithDetails("AWSClientFactoryNewError"))
			},
		},
		{
			caseName: "CloudFormationDescribeStacksError",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}
				dynamicResourceInterfaceMock.On("Namespace", exampleNamespace).Return(dynamicResourceInterfaceMock)
				dynamicResourceInterfaceMock.On("List", mock.Anything, k8smetav1.ListOptions{}).Return(&unstructured.UnstructuredList{Items: exampleUnstructuredList}, (error)(nil))

				dynamicInterfaceMock := &cluster.MockdynamicInterface{}
				dynamicInterfaceMock.On("Resource", exampleSchemaGroupVersionResource).Return(dynamicResourceInterfaceMock)

				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamicInterfaceMock, (error)(nil))

				awsFactoryMock := constructionArguments.awsFactory.(*workflow.MockAWSFactory)
				awsFactoryMock.On("New", functionCallArguments.cluster.OrganizationID, functionCallArguments.cluster.SecretID.ResourceID, functionCallArguments.cluster.Location).Return(exampleAWSClient, (error)(nil))

				stackName := generateNodePoolStackName(functionCallArguments.cluster.Name, exampleNodePoolNames[0])
				describeStacksInput := &cloudformation.DescribeStacksInput{
					StackName: &stackName,
				}

				cloudFormationAPIMock := &workflow.MockcloudFormationAPI{}
				cloudFormationAPIMock.On("DescribeStacks", describeStacksInput).Return(nil, errors.NewWithDetails("CloudFormationDescribeStacksError"))

				cloudFormationFactoryMock := constructionArguments.cloudFormationFactory.(*workflow.MockCloudFormationAPIFactory)
				cloudFormationFactoryMock.On("New", exampleAWSClient).Return(cloudFormationAPIMock)
			},
		},
		{
			caseName: "CloudFormationDescribeStacksNotFoundError",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}
				dynamicResourceInterfaceMock.On("Namespace", exampleNamespace).Return(dynamicResourceInterfaceMock)
				dynamicResourceInterfaceMock.On("List", mock.Anything, k8smetav1.ListOptions{}).Return(&unstructured.UnstructuredList{Items: exampleUnstructuredList}, (error)(nil))

				dynamicInterfaceMock := &cluster.MockdynamicInterface{}
				dynamicInterfaceMock.On("Resource", exampleSchemaGroupVersionResource).Return(dynamicResourceInterfaceMock)

				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamicInterfaceMock, (error)(nil))

				awsFactoryMock := constructionArguments.awsFactory.(*workflow.MockAWSFactory)
				awsFactoryMock.On("New", functionCallArguments.cluster.OrganizationID, functionCallArguments.cluster.SecretID.ResourceID, functionCallArguments.cluster.Location).Return(exampleAWSClient, (error)(nil))

				stackName := generateNodePoolStackName(functionCallArguments.cluster.Name, exampleNodePoolNames[0])
				describeStacksInput := &cloudformation.DescribeStacksInput{
					StackName: &stackName,
				}

				cloudFormationAPIMock := &workflow.MockcloudFormationAPI{}
				cloudFormationAPIMock.On("DescribeStacks", describeStacksInput).Return(&cloudformation.DescribeStacksOutput{}, (error)(nil))

				cloudFormationFactoryMock := constructionArguments.cloudFormationFactory.(*workflow.MockCloudFormationAPIFactory)
				cloudFormationFactoryMock.On("New", exampleAWSClient).Return(cloudFormationAPIMock)
			},
		},
		{
			caseName: "CloudFormationStackParameterError",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}
				dynamicResourceInterfaceMock.On("Namespace", exampleNamespace).Return(dynamicResourceInterfaceMock)
				dynamicResourceInterfaceMock.On("List", mock.Anything, k8smetav1.ListOptions{}).Return(&unstructured.UnstructuredList{Items: exampleUnstructuredList}, (error)(nil))

				dynamicInterfaceMock := &cluster.MockdynamicInterface{}
				dynamicInterfaceMock.On("Resource", exampleSchemaGroupVersionResource).Return(dynamicResourceInterfaceMock)

				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamicInterfaceMock, (error)(nil))

				awsFactoryMock := constructionArguments.awsFactory.(*workflow.MockAWSFactory)
				awsFactoryMock.On("New", functionCallArguments.cluster.OrganizationID, functionCallArguments.cluster.SecretID.ResourceID, functionCallArguments.cluster.Location).Return(exampleAWSClient, (error)(nil))

				stackName := generateNodePoolStackName(functionCallArguments.cluster.Name, exampleNodePoolNames[0])
				describeStacksInput := &cloudformation.DescribeStacksInput{
					StackName: &stackName,
				}

				cloudFormationAPIMock := &workflow.MockcloudFormationAPI{}
				cloudFormationAPIMock.On("DescribeStacks", describeStacksInput).Return(&cloudformation.DescribeStacksOutput{
					Stacks: []*cloudformation.Stack{
						{
							Parameters: []*cloudformation.Parameter{
								{},
							},
						},
					},
				}, (error)(nil))

				cloudFormationFactoryMock := constructionArguments.cloudFormationFactory.(*workflow.MockCloudFormationAPIFactory)
				cloudFormationFactoryMock.On("New", exampleAWSClient).Return(cloudFormationAPIMock)
			},
		},
		{
			caseName: "ListNodePoolsSuccess",
			constructionArguments: constructionArgumentType{
				workflowClient:        exampleWorkflowClient,
				enterprise:            exampleEnterprise,
				dynamicClientFactory:  &cluster.MockDynamicKubeClientFactory{},
				awsFactory:            &workflow.MockAWSFactory{},
				cloudFormationFactory: &workflow.MockCloudFormationAPIFactory{},
				namespace:             exampleNamespace,
			},
			expectedNodePools:   exampleNodePools,
			expectedNotNilError: false,
			functionCallArguments: functionCallArgumentType{
				ctx:           exampleContext,
				cluster:       exampleCluster,
				nodePoolNames: exampleNodePoolNames,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				dynamicResourceInterfaceMock := &cluster.MockdynamicNamespaceableResourceInterface{}
				dynamicResourceInterfaceMock.On("Namespace", exampleNamespace).Return(dynamicResourceInterfaceMock)
				dynamicResourceInterfaceMock.On("List", mock.Anything, k8smetav1.ListOptions{}).Return(&unstructured.UnstructuredList{Items: exampleUnstructuredList}, (error)(nil))

				dynamicInterfaceMock := &cluster.MockdynamicInterface{}
				dynamicInterfaceMock.On("Resource", exampleSchemaGroupVersionResource).Return(dynamicResourceInterfaceMock)

				dynamicClientFactoryMock := constructionArguments.dynamicClientFactory.(*cluster.MockDynamicKubeClientFactory)
				dynamicClientFactoryMock.On("FromSecret", functionCallArguments.ctx, functionCallArguments.cluster.ConfigSecretID.String()).Return(dynamicInterfaceMock, (error)(nil))

				awsFactoryMock := constructionArguments.awsFactory.(*workflow.MockAWSFactory)
				awsFactoryMock.On("New", functionCallArguments.cluster.OrganizationID, functionCallArguments.cluster.SecretID.ResourceID, functionCallArguments.cluster.Location).Return(exampleAWSClient, (error)(nil))

				stackName0 := generateNodePoolStackName(functionCallArguments.cluster.Name, exampleNodePoolNames[0])
				stackName1 := generateNodePoolStackName(functionCallArguments.cluster.Name, exampleNodePoolNames[1])
				describeStacksInput0 := &cloudformation.DescribeStacksInput{
					StackName: &stackName0,
				}
				describeStacksInput1 := &cloudformation.DescribeStacksInput{
					StackName: &stackName1,
				}

				cloudFormationAPIMock := &workflow.MockcloudFormationAPI{}
				cloudFormationAPIMock.On("DescribeStacks", describeStacksInput0).Return(exampleDescribeStacksOutput, (error)(nil))
				cloudFormationAPIMock.On("DescribeStacks", describeStacksInput1).Return(exampleDescribeStacksOutput, (error)(nil))

				cloudFormationFactoryMock := constructionArguments.cloudFormationFactory.(*workflow.MockCloudFormationAPIFactory)
				cloudFormationFactoryMock.On("New", exampleAWSClient).Return(cloudFormationAPIMock)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			testCase.setupMocks(testCase.constructionArguments, testCase.functionCallArguments)

			nodePoolManager := nodePoolManager{
				workflowClient:        testCase.constructionArguments.workflowClient,
				enterprise:            testCase.constructionArguments.enterprise,
				dynamicClientFactory:  testCase.constructionArguments.dynamicClientFactory,
				awsFactory:            testCase.constructionArguments.awsFactory,
				cloudFormationFactory: testCase.constructionArguments.cloudFormationFactory,
				namespace:             testCase.constructionArguments.namespace,
			}

			got, err := nodePoolManager.ListNodePools(
				testCase.functionCallArguments.ctx,
				testCase.functionCallArguments.cluster,
				testCase.functionCallArguments.nodePoolNames,
			)

			require.Truef(t, (err != nil) == testCase.expectedNotNilError,
				"error value doesn't match the expectation, is expected: %+v, actual error value: %+v", testCase.expectedNotNilError, err)
			require.Equal(t, testCase.expectedNodePools, got)
		})
	}
}
