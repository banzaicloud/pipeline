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
	"github.com/banzaicloud/pipeline/internal/cluster/distribution"
	"github.com/banzaicloud/pipeline/internal/global"
)

const (
	eksIAMTemplateName      = "amazon-eks-iam-cf.yaml"
	eksVPCTemplateName      = "amazon-eks-vpc-cf.yaml"
	eksSubnetTemplateName   = "amazon-eks-subnet-cf.yaml"
	eksNodePoolTemplateName = "amazon-eks-nodepool-cf.yaml"
)

var templateBasePath = global.Config.Distribution.EKS.TemplateLocation

// GetVPCTemplate returns the CloudFormation template for creating VPC for EKS cluster
func GetVPCTemplate() (string, error) {
	return distribution.GetCloudFormationTemplate(templateBasePath, eksVPCTemplateName)
}

// GetNodePoolTemplate returns the CloudFormation template for creating node pools for EKS cluster
func GetNodePoolTemplate() (string, error) {
	return distribution.GetCloudFormationTemplate(templateBasePath, eksNodePoolTemplateName)
}

// GetSubnetTemplate returns the CloudFormation template for creating a Subnet
func GetSubnetTemplate() (string, error) {
	return distribution.GetCloudFormationTemplate(templateBasePath, eksSubnetTemplateName)
}

// GetIAMTemplate returns the CloudFormation template for creating IAM roles for the EKS cluster
func GetIAMTemplate() (string, error) {
	return distribution.GetCloudFormationTemplate(templateBasePath, eksIAMTemplateName)
}
