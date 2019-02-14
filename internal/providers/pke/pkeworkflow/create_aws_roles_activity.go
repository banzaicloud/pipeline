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
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const CreateAWSRolesActivityName = "pke-create-aws-roles-activity"

type CreateAWSRolesActivity struct {
	clusters Clusters
}

func NewCreateAWSRolesActivity(clusters Clusters) *CreateAWSRolesActivity {
	return &CreateAWSRolesActivity{
		clusters: clusters,
	}
}

type CreateAWSRolesActivityInput struct {
	ClusterID uint
}

func (a *CreateAWSRolesActivity) Execute(ctx context.Context, input CreateAWSRolesActivityInput) (string, error) {
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return "", errors.New(fmt.Sprintf("can't create AWS roles for %t", cluster))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", emperror.Wrap(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	// check if global roles are already created for another cluster
	stackFilter := cloudformation.ListStacksInput{
		StackStatusFilter: aws.StringSlice([]string{"CREATE_COMPLETE", "CREATE_IN_PROGRESS"}),
	}

	// TODO: remove this and replace with CreateStack -> ErrCodeAlreadyExistsException handling
	for {
		stacks, err := cfClient.ListStacks(&stackFilter)
		if err != nil {
			return "", emperror.Wrap(err, "failed to check if role already exists")
		}

		for _, stack := range stacks.StackSummaries {
			if *stack.StackName == "pke-global" {
				if *stack.StackStatus == "CREATE_IN_PROGRESS" {
					return *stack.StackId, nil
				}
				return "", nil
			}
		}

		if stacks.NextToken != nil {
			stackFilter = cloudformation.ListStacksInput{NextToken: stacks.NextToken}
		} else {
			break
		}
	}

	buf, err := ioutil.ReadFile("templates/global.cf.tpl")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}

	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}),
		StackName:    aws.String("pke-global"),
		TemplateBody: aws.String(string(buf)),
	}

	output, err := cfClient.CreateStack(stackInput)
	if err != nil {
		return "", emperror.Wrap(err, "creating role")
	}

	return *output.StackId, nil
}
