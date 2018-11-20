// Copyright © 2018 Banzai Cloud
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

package errors

import "errors"

// ### [ Errors ] ### //
var (
	ErrorNotSupportedCloudType    = errors.New("Not supported cloud type")
	ErrorAmazonImageFieldIsEmpty  = errors.New("Required field 'image' is empty ")
	ErrorInstancetypeFieldIsEmpty = errors.New("Required field 'instanceType' is empty ")

	ErrorAmazonEksFieldIsEmpty         = errors.New("Required field 'eks' is empty.")
	ErrorAmazonEksNodePoolFieldIsEmpty = errors.New("At least one 'nodePool' is required.")

	ErrorNodePoolMinMaxFieldError     = errors.New("'maxCount' must be greater than 'minCount'")
	ErrorNodePoolCountFieldError      = errors.New("'count' must be greater than or equal to 'minCount' and lower than or equal to 'maxCount'")
	ErrorMinFieldRequiredError        = errors.New("'minCount' must be set in case 'autoscaling' is set to true")
	ErrorMaxFieldRequiredError        = errors.New("'maxCount' must be set in case 'autoscaling' is set to true")
	ErrorAzureFieldIsEmpty            = errors.New("Azure is <nil>")
	ErrorNodePoolEmpty                = errors.New("Required field 'nodePools' is empty.")
	ErrorNotDifferentInterfaces       = errors.New("There is no change in data")
	ErrorClusterNotReady              = errors.New("Cluster not ready yet")
	ErrorNilCluster                   = errors.New("<nil> cluster")
	ErrorWrongKubernetesVersion       = errors.New("Wrong kubernetes version for master/nodes. The required minimum kubernetes version is 1.8.x ")
	ErrorDifferentKubernetesVersion   = errors.New("Different kubernetes version for master and nodes")
	ErrorLocationEmpty                = errors.New("Location field is empty")
	ErrorRequiredLocation             = errors.New("location is required")
	ErrorRequiredSecretId             = errors.New("Secret id is required")
	ErrorCloudInfoK8SNotSupported     = errors.New("Not supported key in case of amazon")
	ErrorNodePoolNotProvided          = errors.New("At least one 'nodepool' is required for creating or updating a cluster")
	ErrorNotValidLocation             = errors.New("not valid location")
	ErrorNotValidNodeImage            = errors.New("not valid node image")
	ErrorNotValidNodeInstanceType     = errors.New("not valid nodeInstanceType")
	ErrorNotValidMasterVersion        = errors.New("not valid master version")
	ErrorNotValidNodeVersion          = errors.New("not valid node version")
	ErrorNotValidKubernetesVersion    = errors.New("not valid kubernetesVersion")
	ErrorResourceGroupRequired        = errors.New("resource group is required")
	ErrStateStorePathEmpty            = errors.New("statestore path cannot be empty")
	ErrorAlibabaFieldIsEmpty          = errors.New("Required field 'alibaba' is empty.")
	ErrorAlibabaRegionIDFieldIsEmpty  = errors.New("Required field 'region_id' is empty.")
	ErrorAlibabaZoneIDFieldIsEmpty    = errors.New("Required field 'zoneid' is empty.")
	ErrorAlibabaNodePoolFieldIsEmpty  = errors.New("At least one 'nodePool' is required.")
	ErrorAlibabaNodePoolFieldLenError = errors.New("Only one 'nodePool' is supported.")
	ErrorAlibabaMinNumberOfNodes      = errors.New("'num_of_nodes' must be greater than zero.")
	ErrorBucketDeleteNotEmpty         = errors.New("failed to delete bucket, it was not empty")
)
