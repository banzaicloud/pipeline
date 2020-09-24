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
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
	"github.com/banzaicloud/pipeline/internal/providers/amazon"
)

const CreateMasterActivityName = "pke-create-master-activity"

type CreateMasterActivity struct {
	clusters       Clusters
	tokenGenerator TokenGenerator
}

func NewCreateMasterActivity(
	clusters Clusters,
	tokenGenerator TokenGenerator,
) *CreateMasterActivity {
	return &CreateMasterActivity{
		clusters:       clusters,
		tokenGenerator: tokenGenerator,
	}
}

type CreateMasterActivityInput struct {
	ClusterID                 uint
	VPCID                     string
	VPCDefaultSecurityGroupID string
	SubnetIDs                 []string
	MultiMaster               bool
	MasterInstanceProfile     string
	ExternalBaseUrl           string
	ExternalBaseUrlInsecure   bool
	Pool                      NodePool
	SSHKeyName                string

	EIPAllocationID string

	TargetGroup string
}

func (a *CreateMasterActivity) Execute(ctx context.Context, input CreateMasterActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return "", errors.New(fmt.Sprintf("can't create VPC for cluster type %t", cluster))
	}

	_, signedToken, err := a.tokenGenerator.GenerateClusterToken(cluster.GetOrganizationId(), cluster.GetID())
	if err != nil {
		return "", errors.WrapIf(err, "can't generate Pipeline token")
	}

	bootstrapCommand, err := awsCluster.GetBootstrapCommand("master", input.ExternalBaseUrl, input.ExternalBaseUrlInsecure, signedToken, nil)
	if err != nil {
		return "", errors.WrapIf(err, "failed to fetch bootstrap command")
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", errors.WrapIf(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	target := "master"
	if input.MultiMaster {
		target = "masters"
	}

	buf, err := ioutil.ReadFile(fmt.Sprintf("templates/pke/%s.cf.yaml", target))
	if err != nil {
		return "", errors.WrapIf(err, "loading CF template")
	}
	clusterName := cluster.GetName()

	params := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: &clusterName,
		},
		{
			ParameterKey:   aws.String("PkeCommand"),
			ParameterValue: &bootstrapCommand,
		},
		{
			ParameterKey:   aws.String("InstanceType"),
			ParameterValue: aws.String(input.Pool.InstanceType),
		},
		{
			ParameterKey:   aws.String("VPCId"),
			ParameterValue: &input.VPCID,
		},
		{
			ParameterKey:   aws.String("VPCDefaultSecurityGroupId"),
			ParameterValue: &input.VPCDefaultSecurityGroupID,
		},
		{
			ParameterKey:   aws.String("PkeCommand"),
			ParameterValue: &bootstrapCommand,
		},
		{
			ParameterKey:   aws.String("IamInstanceProfile"),
			ParameterValue: &input.MasterInstanceProfile,
		},
		{
			ParameterKey:   aws.String("ImageId"),
			ParameterValue: aws.String(input.Pool.ImageID),
		},
		{
			ParameterKey:   aws.String("VolumeSize"),
			ParameterValue: aws.String(strconv.Itoa(input.Pool.VolumeSize)),
		},
		{
			ParameterKey:   aws.String("PkeVersion"),
			ParameterValue: aws.String(pkeaws.Version),
		},
		{
			ParameterKey:   aws.String("KeyName"),
			ParameterValue: aws.String(input.SSHKeyName),
		},
	}

	stackName := fmt.Sprintf("pke-master-%s", clusterName)

	if input.MultiMaster {
		params = append(params,
			[]*cloudformation.Parameter{
				{
					ParameterKey:   aws.String("TargetGroup"),
					ParameterValue: aws.String(input.TargetGroup),
				}, {
					ParameterKey:   aws.String("SubnetIds"),
					ParameterValue: aws.String(strings.Join(input.SubnetIDs, ",")),
				},
			}...)
	} else {
		params = append(params,
			[]*cloudformation.Parameter{
				{
					ParameterKey:   aws.String("EIPAllocationId"),
					ParameterValue: aws.String(input.EIPAllocationID),
				}, {
					ParameterKey:   aws.String("SubnetId"),
					ParameterValue: aws.String(input.SubnetIDs[0]),
				},
			}...)
	}

	stackInput := &cloudformation.CreateStackInput{
		Capabilities: aws.StringSlice([]string{cloudformation.CapabilityCapabilityAutoExpand}),
		StackName:    &stackName,
		TemplateBody: aws.String(string(buf)),
		Parameters:   params,
		Tags:         amazon.PipelineTags(),
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
