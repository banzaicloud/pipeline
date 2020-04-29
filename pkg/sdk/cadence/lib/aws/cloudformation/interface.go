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

package cloudformation

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/workflow"
)

// CloudFormation implements AWS CF operations and wraps them in a Cadence specific context.
type CloudFormation interface {
	// WaitUntilStackUpdateComplete implements the CF operation with the same name.
	WaitUntilStackUpdateComplete(ctx workflow.Context, input *cloudformation.DescribeStacksInput) error
}
