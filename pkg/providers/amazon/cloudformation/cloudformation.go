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

package cloudformation

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

// GetExistingTaggedStackNames gives back existing CF stacks which have the given tags
func GetExistingTaggedStackNames(cfSvc *cloudformation.CloudFormation, tags map[string]string) ([]string, error) {
	names := make([]string, 0)

	err := cfSvc.DescribeStacksPages(&cloudformation.DescribeStacksInput{}, func(page *cloudformation.DescribeStacksOutput, lastPage bool) bool {
		for _, stack := range page.Stacks {
			stackTags := getFlattenedTags(stack.Tags)
			ok := true
			for k, v := range tags {
				if stackTags[k] != v {
					ok = false
					break
				}
			}
			if ok && stack.StackName != nil {
				names = append(names, *stack.StackName)
			}
		}
		return true
	})

	return names, err
}

func getFlattenedTags(tags []*cloudformation.Tag) map[string]string {

	t := make(map[string]string, 0)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			t[*tag.Key] = *tag.Value
		}
	}

	return t
}
