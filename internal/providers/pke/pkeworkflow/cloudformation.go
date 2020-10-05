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

package pkeworkflow

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"

	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
)

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

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
