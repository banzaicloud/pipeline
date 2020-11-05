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
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/internal/global"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const GetSubnetStacksActivityName = "eks-get-subnet-stacks"

// GetSubnetStacksActivity collects all subnet stack names
type GetSubnetStacksActivity struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
}

type GetSubnetStacksActivityInput struct {
	EKSActivityInput
}

type GetSubnetStacksActivityOutput struct {
	StackNames []string
}

func NewGetSubnetStacksActivity(awsSessionFactory *awsworkflow.AWSSessionFactory) *GetSubnetStacksActivity {
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

// getClusterSubnetStackNames returns the names of the CloudFormation stacks of
// the specified cluster.
//
// This is a convenience wrapper around the corresponding activity.
func getClusterSubnetStackNames(ctx workflow.Context, eksActivityInput EKSActivityInput) ([]string, error) {
	var activityOutput GetSubnetStacksActivityOutput
	err := getClusterSubnetStackNamesAsync(ctx, eksActivityInput).Get(ctx, &activityOutput)
	if err != nil {
		return nil, err
	}

	return activityOutput.StackNames, nil
}

// getClusterSubnetStackNamesAsync returns a future object retrieving the names
// of the CloudFormation stacks of the specified cluster.
//
// This is a convenience wrapper around the corresponding activity.
func getClusterSubnetStackNamesAsync(ctx workflow.Context, eksActivityInput EKSActivityInput) workflow.Future {
	return workflow.ExecuteActivity(ctx, GetSubnetStacksActivityName, GetSubnetStacksActivityInput{
		EKSActivityInput: eksActivityInput,
	})
}
