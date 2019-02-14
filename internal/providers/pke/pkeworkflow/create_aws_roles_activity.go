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
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
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
	awsCluster := cluster.(AWSCluster)
	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", emperror.Wrap(err, "failed to connect to AWS")
	}

	cloudformationSrv := cloudformation.New(client)

	if ok, err := CheckPkeGlobalCF(cloudformationSrv); err != nil {
		return "", emperror.Wrap(err, "checking if role exists")
	} else if ok {
		// already exists
		// TODO: move check out of this action
		// TODO: wait for completion of existing stack
		return "", nil
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
	output, err := cloudformationSrv.CreateStack(stackInput)
	if err != nil {
		return "", emperror.Wrap(err, "creating role")
	}

	return *output.StackId, nil
}

// CheckPkeGlobalCF returns if global roles are already created by us for an other cluster
func CheckPkeGlobalCF(cloudformationSrv *cloudformation.CloudFormation) (bool, error) {
	stackFilter := cloudformation.ListStacksInput{
		StackStatusFilter: aws.StringSlice([]string{"CREATE_COMPLETE", "CREATE_IN_PROGRESS"}),
	}
	stacks, err := cloudformationSrv.ListStacks(&stackFilter)
	if err != nil {
		return false, err
	}
	for _, stack := range stacks.StackSummaries {
		if *stack.StackName == "pke-global" {
			return true, nil
		}
	}
	return false, nil
}
