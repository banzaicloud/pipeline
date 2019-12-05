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
	"context"

	"emperror.dev/errors"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/global"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const GetSubnetStacksActivityName = "eks-get-subnet-stacks"

// GetSubnetStacksActivity collects all subnet stack names
type GetSubnetStacksActivity struct {
	awsSessionFactory *AWSSessionFactory
}

type GetSubnetStacksActivityInput struct {
	EKSActivityInput
}

type GetSubnetStacksActivityOutput struct {
	StackNames []string
}

func NewGetSubnetStacksActivity(awsSessionFactory *AWSSessionFactory) *GetSubnetStacksActivity {
	return &GetSubnetStacksActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *GetSubnetStacksActivity) Execute(ctx context.Context, input GetSubnetStacksActivityInput) (*GetSubnetStacksActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"region", input.Region,
		"cluster", input.ClusterName,
	)

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	tags := map[string]string{
		global.ManagedByPipelineTag:         global.ManagedByPipelineValue,
		"banzaicloud-pipeline-cluster-name": input.ClusterName,
		"banzaicloud-pipeline-stack-type":   "subnet",
	}

	cfStackNames, err := pkgCloudformation.GetExistingTaggedStackNames(cloudformation.New(session), tags)
	if err != nil {
		return nil, err
	}

	logger.Infof("Subnets used by cluster: '%s'", cfStackNames)
	output := GetSubnetStacksActivityOutput{
		StackNames: cfStackNames,
	}

	return &output, nil
}
