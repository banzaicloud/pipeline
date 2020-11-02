// Copyright © 2019 Banzai Cloud
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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

// +testify:mock:testOnly=true

// ec2API redefines the ec2iface.EC2API interface in order to generate mock for
// it.
//
// nolint:deadcode // Used for mock generation and only the original interface
// is referenced.
type ec2API interface {
	ec2iface.EC2API
}

// +testify:mock:testOnly=true

// EC2APIFactory provides an interface for instantiating AWS EC2 API objects.
type EC2APIFactory interface {
	// New instantiates an AWS CloudFormation API object based on the specified
	// configurations.
	New(configProvider client.ConfigProvider, configs ...*aws.Config) (ec2API ec2iface.EC2API)
}

// EC2Factory can instantiate am ec2.EC2 object.
//
// Implements the pkeworkflow.EC2APIFactory interface.
type EC2Factory struct{}

// NewEC2Factory instantiates an EC2Factory object.
func NewEC2Factory() (factory *EC2Factory) {
	return &EC2Factory{}
}

// New instantiates an AWS EC2 API object based on the specified configurations.
//
// Implements the pkeworkflow.EC2APIFactory interface.
func (factory *EC2Factory) New(
	configProvider client.ConfigProvider,
	configs ...*aws.Config,
) (ec2API ec2iface.EC2API) {
	return ec2.New(configProvider, configs...)
}
