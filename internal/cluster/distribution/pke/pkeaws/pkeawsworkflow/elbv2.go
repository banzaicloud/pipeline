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

package pkeawsworkflow

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

// +testify:mock:testOnly=true

// cloudFormationAPI redefines the elbv2iface.ELBV2API
// interface in order to generate mock for it.
// nolint:deadcode // Used for mock generation and only the original interface
// is referenced.
type elbv2API interface {
	elbv2iface.ELBV2API
}

// +testify:mock:testOnly=true

// ELBV2APIFactory provides an interface for instantiating AWS ELBV2 API objects.
type ELBV2APIFactory interface {
	// New instantiates an AWS ELBV2 API object based on the specified
	// configurations.
	New(configProvider client.ConfigProvider, configs ...*aws.Config) (elbv2API elbv2iface.ELBV2API)
}

// ELBV2Factory can instantiate am elbv2.ELBV2 object.
//
// Implements the ELBV2APIFactory interface.
type ELBV2Factory struct{}

// NewELBV2Factory instantiates an pkeawsworkflow.ELBV2Factory object.
func NewELBV2Factory() (factory *ELBV2Factory) {
	return &ELBV2Factory{}
}

// New instantiates an AWS ELBV2 API object based on the specified configurations.
//
// Implements the pkeawsworkflow.ELBV2Factory interface.
func (factory *ELBV2Factory) New(
	configProvider client.ConfigProvider,
	configs ...*aws.Config,
) (elbv2API elbv2iface.ELBV2API) {
	return elbv2.New(configProvider, configs...)
}
