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

	"github.com/banzaicloud/pipeline/internal/global"

	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const GetNodepoolStacksActivityName = "eks-get-nodepool-stacks"

type GetNodepoolStacksActivity struct {
	awsSessionFactory *AWSSessionFactory
}

type GetNodepoolStacksActivityInput struct {
	EKSActivityInput
	NodePoolNames []string
}

type GetNodepoolStacksActivityOutput struct {
	StackNames []string
}

func NewGetNodepoolStacksActivity(awsSessionFactory *AWSSessionFactory) *GetNodepoolStacksActivity {
	return &GetNodepoolStacksActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *GetNodepoolStacksActivity) Execute(ctx context.Context, input GetNodepoolStacksActivityInput) (*GetNodepoolStacksActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	stackNames := make([]string, 0)
	uniqueMap := make(map[string]bool, 0)

	for _, nodePool := range input.NodePoolNames {
		nodePoolStackName := GenerateNodePoolStackName(input.ClusterName, nodePool)
		stackNames = append(stackNames, nodePoolStackName)
		uniqueMap[nodePoolStackName] = true
	}

	logger.Debugf("stack names from DB: %+v", stackNames)

	oldTags := map[string]string{
		"pipeline-created":      "true",
		"pipeline-cluster-name": input.ClusterName,
		"pipeline-stack-type":   "nodepool",
	}
	tags := map[string]string{
		global.ManagedByPipelineTag:         global.ManagedByPipelineValue,
		"banzaicloud-pipeline-cluster-name": input.ClusterName,
		"banzaicloud-pipeline-stack-type":   "nodepool",
	}

	// for backward compatibility looks for node pool stacks tagged by earlier version of Pipeline
	cfStackNamesByOldTags, err := pkgCloudformation.GetExistingTaggedStackNames(cloudformation.New(awsSession), oldTags)
	if err != nil {
		return nil, err
	}

	cfStackNames, err := pkgCloudformation.GetExistingTaggedStackNames(cloudformation.New(awsSession), tags)
	if err != nil {
		return nil, err
	}
	cfStackNames = append(cfStackNames, cfStackNamesByOldTags...)

	for _, stackName := range cfStackNames {
		if !uniqueMap[stackName] {
			stackNames = append(stackNames, stackName)
		}
	}

	logger.Debugf("stack names from DB + CF: %+v", stackNames)

	output := GetNodepoolStacksActivityOutput{
		StackNames: stackNames,
	}
	return &output, nil
}
