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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence/activity"
)

const CreateAWSRolesActivityName = "pke-create-aws-roles-activity"
const PkeGlobalStackName = "pke-global"

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
	Region    string
}

func (a *CreateAWSRolesActivity) Execute(ctx context.Context, input CreateAWSRolesActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
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

	// TODO maybe check for roles not stack, now overriding region to avoid collision
	client.Config.Region = &input.Region

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/global.cf.yaml")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}

	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityIam, cloudformation.CapabilityCapabilityNamedIam}),
		StackName:    aws.String(PkeGlobalStackName),
		TemplateBody: aws.String(string(buf)),
	}

	output, err := cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			log.Infof("stack already exists: %s", err.Message())
			return PkeGlobalStackName, nil
		default:
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return *output.StackId, nil
}
