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

package awsworkflow

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cfi "github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

// +testify:mock

// cloudFormationAPI redefines the cloudformationiface.CloudFormationAPI
// interface in order to generate mock for it.
// nolint:deadcode // Used for mock generation and only the original interface
// is referenced.
type cloudFormationAPI interface {
	cfi.CloudFormationAPI
}

// +testify:mock

// CloudFormationAPIFactory provides an interface for instantiating AWS
// CloudFormation API objects.
type CloudFormationAPIFactory interface {
	// New instantiates an AWS CloudFormation API object based on the specified
	// configurations.
	New(configProvider client.ConfigProvider, configs ...*aws.Config) (cloudFormationAPI cfi.CloudFormationAPI)
}

// CloudFormationFactory can instantiate a cloudformation.CloudFormation object.
// Implements the workflow.CloudFormationAPIFactory interface.
type CloudFormationFactory struct{}

// NewCloudFormationFactory instantiates a CloudFormationFactory object.
func NewCloudFormationFactory() (factory *CloudFormationFactory) {
	return &CloudFormationFactory{}
}

// New instantiates an AWS CloudFormation API object based on the specified
// configurations.
// Implements the workflow.CloudFormationAPIFactory interface.
func (factory *CloudFormationFactory) New(
	configProvider client.ConfigProvider,
	configs ...*aws.Config,
) (cloudFormationAPI cfi.CloudFormationAPI) {
	return cloudformation.New(configProvider, configs...)
}
