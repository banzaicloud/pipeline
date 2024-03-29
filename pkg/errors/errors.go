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
	ErrorNotSupportedCloudType        = errors.New("Not supported cloud type")
	ErrorNotSupportedDistributionType = errors.New("Not supported distribution type")
	ErrorAmazonImageFieldIsEmpty      = errors.New("Required field 'image' is empty ")
	ErrorInstancetypeFieldIsEmpty     = errors.New("Required field 'instanceType' is empty ")

	ErrorAmazonEksFieldIsEmpty         = errors.New("Required field 'eks' is empty.")
	ErrorAmazonEksNodePoolFieldIsEmpty = errors.New("At least one 'nodePool' is required.")

	ErrorNodePoolMinMaxFieldError      = errors.New("'maxCount' must be greater than 'minCount'")
	ErrorNodePoolCountFieldError       = errors.New("'count' must be greater than or equal to 'minCount' and lower than or equal to 'maxCount'")
	ErrorMaxFieldRequiredError         = errors.New("'maxCount' must be set in case 'autoscaling' is set to true")
	ErrorAzureFieldIsEmpty             = errors.New("Azure is <nil>")
	ErrorNodePoolEmpty                 = errors.New("Required field 'nodePools' is empty.")
	ErrorNotDifferentInterfaces        = errors.New("There is no change in data")
	ErrorNilCluster                    = errors.New("<nil> cluster")
	ErrorWrongKubernetesVersion        = errors.New("Wrong kubernetes version for master/nodes. The required minimum kubernetes version is 1.8.x ")
	ErrorNotSupportedKubernetesVersion = errors.New("Not supported Kubernetes version")
	ErrorDifferentKubernetesVersion    = errors.New("Different kubernetes version for master and nodes")
	ErrorLocationEmpty                 = errors.New("Location field is empty")
	ErrorNodePoolNotProvided           = errors.New("At least one 'nodepool' is required for creating or updating a cluster")
	ErrorNotValidLocation              = errors.New("not valid location")
	ErrorNotValidNodeInstanceType      = errors.New("not valid nodeInstanceType")
	ErrorNotValidMasterVersion         = errors.New("not valid master version")
	ErrorNotValidNodeVersion           = errors.New("not valid node version")
	ErrorNotValidKubernetesVersion     = errors.New("not valid kubernetesVersion")
	ErrorResourceGroupRequired         = errors.New("resource group is required")
	ErrorBucketDeleteNotEmpty          = errors.New("non empty buckets can not be deleted")
	ErrorGkeSubnetRequiredFieldIsEmpty = errors.New("'subnet' field required if 'vpc' is set")
	ErrorGkeVPCRequiredFieldIsEmpty    = errors.New("'vpc' field required if 'subnet' is set")

	ErrorMissingCloudSpecificProperties = errors.New("missing cloud specific properties")
)

// BadRequestBehavior can be used to add the BadRequest() bool behavior to error implementations.
type BadRequestBehavior struct{}

// BadRequest returns true.
func (BadRequestBehavior) BadRequest() bool {
	return true
}

// ClientErrorBehavior can be used to add the ClientError() bool behavior to error implementations.
type ClientErrorBehavior struct{}

// ClientError returns true.
func (ClientErrorBehavior) ClientError() bool {
	return true
}

// ValidationBehavior can be used to add the Validation() bool behavior to error implementations.
type ValidationBehavior struct{}

// Validation returns true.
func (ValidationBehavior) Validation() bool {
	return true
}
