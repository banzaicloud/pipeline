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

package pkeworkflow

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
)

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

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

// getStackTags returns the tags that are placed onto CF template stacks.
// These tags  are propagated onto the resources created by the CF template.
func getStackTags(clusterName, stackType string, clusterTags map[string]string) []*cloudformation.Tag {
	tags := make([]*cloudformation.Tag, 0)

	for k, v := range clusterTags {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	tags = append(tags, []*cloudformation.Tag{
		{Key: aws.String("banzaicloud-pipeline-cluster-name"), Value: aws.String(clusterName)},
		{Key: aws.String("banzaicloud-pipeline-stack-type"), Value: aws.String(stackType)},
	}...)
	tags = append(tags, internalAmazon.PipelineTags()...)
	return tags
}

func getNodePoolStackTags(clusterName string, clusterTags map[string]string) []*cloudformation.Tag {
	return getStackTags(clusterName, "nodepool", clusterTags)
}

func getSubnetStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "subnet", map[string]string{})
}
