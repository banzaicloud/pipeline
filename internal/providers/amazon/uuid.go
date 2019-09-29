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

package amazon

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"

	"github.com/banzaicloud/pipeline/internal/global"
)

// PipelineTags returns resource tags for azure based on the pipeline uuid if available
func PipelineTags() []*cloudformation.Tag {
	tags := []*cloudformation.Tag{
		{
			Key:   aws.String(global.ManagedByPipelineTag),
			Value: aws.String(global.ManagedByPipelineValue),
		},
	}

	value := global.PipelineUUID()
	if value != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(global.ManagedByPipelineUUIDTag),
			Value: aws.String(value),
		})
	}

	return tags
}
