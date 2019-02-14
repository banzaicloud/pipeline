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
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const WaitCFCompletionActivityName = "pke-wait-cf-completion-activity"

type WaitCFCompletionActivity struct {
	clusters Clusters
}

func NewWaitCFCompletionActivity(clusters Clusters) *WaitCFCompletionActivity {
	return &WaitCFCompletionActivity{
		clusters: clusters,
	}
}

type WaitCFCompletionActivityInput struct {
	ClusterID uint
	StackID   string
}

func (a *WaitCFCompletionActivity) Execute(ctx context.Context, input WaitCFCompletionActivityInput) (map[string]string, error) {
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return nil, err
	}

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return nil, errors.New(fmt.Sprintf("can't wait for AWS roles for %t", cluster))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackID)}

	err = cfClient.WaitUntilStackCreateCompleteWithContext(ctx, describeStacksInput)
	if err != nil {
		return nil, emperror.Wrap(err, "error waiting Cloud Formation template")
	}
	output, err := cfClient.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, emperror.Wrap(err, "error fetching Cloud Formation template")
	}
	outputMap := map[string]string{}
	for _, p := range output.Stacks[0].Outputs {
		outputMap[*p.OutputKey] = *p.OutputValue
	}
	return outputMap, nil
}
