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

package workflow

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/eks"
	eksi "github.com/aws/aws-sdk-go/service/eks/eksiface"
)

// +testify:mock

// eksAPI redefines the eksiface.EKSAPI
// interface in order to generate mock for it.
// nolint:deadcode // Used for mock generation and only the original interface
// is referenced.
type eksAPI interface {
	eksi.EKSAPI
}

// +testify:mock

// EKSAPIFactory provides an interface for instantiating AWS
// EKS API objects.
type EKSAPIFactory interface {
	// New instantiates an AWS EKS API object based on the specified
	// configurations.
	New(configProvider client.ConfigProvider, configs ...*aws.Config) (eksAPI eksi.EKSAPI)
}

// EKSFactory can instantiate a eks.EKS object.
// Implements the workflow.EKSAPIFactory interface.
type EKSFactory struct{}

// NewEKSFactory instantiates a EKSFactory object.
func NewEKSFactory() EKSAPIFactory {
	return &EKSFactory{}
}

// New instantiates an AWS EKS API object based on the specified
// configurations.
// Implements the workflow.EKSAPIFactory interface.
func (factory *EKSFactory) New(
	configProvider client.ConfigProvider,
	configs ...*aws.Config,
) (eksAPI eksi.EKSAPI) {
	return eks.New(configProvider, configs...)
}
