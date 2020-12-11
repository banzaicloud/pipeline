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
	internalcloudformation "github.com/banzaicloud/pipeline/internal/cloudformation"
	"github.com/banzaicloud/pipeline/internal/global"
)

const (
	eksIAMTemplateName      = "amazon-eks-iam-cf.yaml"
	eksVPCTemplateName      = "amazon-eks-vpc-cf.yaml"
	eksSubnetTemplateName   = "amazon-eks-subnet-cf.yaml"
	eksNodePoolTemplateName = "amazon-eks-nodepool-cf.yaml"
)

// GetVPCTemplate returns the CloudFormation template for creating VPC for EKS cluster
func GetVPCTemplate() (string, error) {
	return internalcloudformation.GetCloudFormationTemplate(
		global.Config.Distribution.EKS.TemplateLocation, eksVPCTemplateName,
	)
}

// GetNodePoolTemplate returns the CloudFormation template for creating node pools for EKS cluster
func GetNodePoolTemplate() (string, error) {
	return internalcloudformation.GetCloudFormationTemplate(
		global.Config.Distribution.EKS.TemplateLocation, eksNodePoolTemplateName,
	)
}

// GetSubnetTemplate returns the CloudFormation template for creating a Subnet
func GetSubnetTemplate() (string, error) {
	return internalcloudformation.GetCloudFormationTemplate(
		global.Config.Distribution.EKS.TemplateLocation, eksSubnetTemplateName,
	)
}

// GetIAMTemplate returns the CloudFormation template for creating IAM roles for the EKS cluster
func GetIAMTemplate() (string, error) {
	return internalcloudformation.GetCloudFormationTemplate(
		global.Config.Distribution.EKS.TemplateLocation, eksIAMTemplateName,
	)
}
