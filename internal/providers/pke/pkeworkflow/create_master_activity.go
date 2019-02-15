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
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
	"io/ioutil"
)

const CreateMasterActivityName = "pke-create-master-activity"

type CreateMasterActivity struct {
	clusters       Clusters
	tokenGenerator TokenGenerator
}

func NewCreateMasterActivity(clusters Clusters, tokenGenerator TokenGenerator) *CreateMasterActivity {
	return &CreateMasterActivity{
		clusters:       clusters,
		tokenGenerator: tokenGenerator,
	}
}

type CreateMasterActivityInput struct {
	ClusterID uint
}

func (a *CreateMasterActivity) Execute(ctx context.Context, input CreateMasterActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return "", errors.New(fmt.Sprintf("can't create VPC for cluster type %t", c))
	}

	_, signedToken, err := a.tokenGenerator.GenerateClusterToken(c.GetOrganizationId(), c.GetID())
	if err != nil {
		return "", emperror.Wrap(err, "can't generate Pipeline token")
	}

	bootstrapCommand, err := awsCluster.GetBootstrapCommand("master", "url", signedToken)

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", emperror.Wrap(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/master.cf.yaml")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}
	clusterName := c.GetName()
	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityAutoExpand}),
		StackName:    aws.String("pke-master-" + c.GetName()),
		TemplateBody: aws.String(string(buf)),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("ClusterName"),
				ParameterValue: &clusterName,
			},
			{
				ParameterKey:   aws.String("PkeCommand"),
				ParameterValue: &bootstrapCommand,
			},
		},
	}

	output, err := cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			log.Infof("stack already exists: %s", err.Message())
		default:
			return "", err
		}
	}
	return *output.StackId, nil
}
