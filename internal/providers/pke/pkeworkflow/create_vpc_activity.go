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

const CreateVPCActivityName = "pke-create-vpc-activity"

type CreateVPCActivity struct {
	clusters Clusters
}

func NewCreateVPCActivity(clusters Clusters) *CreateVPCActivity {
	return &CreateVPCActivity{
		clusters: clusters,
	}
}

type CreateVPCActivityInput struct {
	ClusterID uint
}

func (a *CreateVPCActivity) Execute(ctx context.Context, input CreateVPCActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return "", errors.New(fmt.Sprintf("can't create VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", emperror.Wrap(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/vpc.cf.yaml")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}
	clusterName := c.GetName()
	stackName := "pke-vpc-" + clusterName
	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityAutoExpand}),
		StackName:    &stackName,
		TemplateBody: aws.String(string(buf)),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:     aws.String("ClusterName"),
				ParameterValue:   &clusterName,
			},
		},
	}

	output, err := cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			log.Infof("stack already exists: %s", err.Message())
		    return stackName, nil
		default:
			return "", err
		}
	}
	return *output.StackId, nil
}
