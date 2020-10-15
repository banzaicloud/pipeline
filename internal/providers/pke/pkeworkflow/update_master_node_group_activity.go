// Copyright © 2020 Banzai Cloud
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
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"

	cloudformation2 "github.com/banzaicloud/pipeline/internal/cloudformation"
	pkgCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	pkgCloudFormation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
)

const UpdateMasterNodeGroupActivityName = "pke-aws-update-master-node-group"

// UpdateMasterNodeGroupActivity updates a master node group.
type UpdateMasterNodeGroupActivity struct {
	sessionFactory AWSFactory
	clusters       Clusters
	tokenGenerator TokenGenerator

	externalBaseURL         string
	externalBaseURLInsecure bool
}

// UpdateMasterNodeGroupActivityInput holds the parameters for the node group update.
type UpdateMasterNodeGroupActivityInput struct {
	SecretID string
	Region   string

	OrganizationID uint

	ClusterName string
	ClusterID   uint

	StackName string

	NodePoolName    string
	NodePoolVersion string

	NodeVolumeSize  int
	NodeImage       string
	Version         string
	DesiredCapacity int64

	ClusterTags map[string]string
}

type UpdateMasterNodeGroupActivityOutput struct {
	NodePoolChanged bool
}

// NewUpdateMasterNodeGroupActivity creates a new UpdateMasterNodeGroupActivity instance.
func NewUpdateMasterNodeGroupActivity(
	sessionFactory AWSFactory,
	clusters Clusters,
	tokenGenerator TokenGenerator,
	externalBaseURL string,
	externalBaseURLInsecure bool,
) UpdateMasterNodeGroupActivity {
	return UpdateMasterNodeGroupActivity{
		sessionFactory:          sessionFactory,
		clusters:                clusters,
		tokenGenerator:          tokenGenerator,
		externalBaseURL:         externalBaseURL,
		externalBaseURLInsecure: externalBaseURLInsecure,
	}
}

// Register registers the activity in the worker.
func (a UpdateMasterNodeGroupActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: UpdateMasterNodeGroupActivityName})
}

// Execute is the main body of the activity, returns true if there was any update and that was successful.
func (a UpdateMasterNodeGroupActivity) Execute(ctx context.Context, input UpdateMasterNodeGroupActivityInput) (UpdateMasterNodeGroupActivityOutput, error) {
	providerSecret, err := brn.Parse(input.SecretID)
	if err != nil {
		return UpdateMasterNodeGroupActivityOutput{}, errors.WrapIf(err, "failed to parse secret BRN")
	}
	sess, err := a.sessionFactory.New(providerSecret.OrganizationID, providerSecret.ResourceID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil { // internal error?
		return UpdateMasterNodeGroupActivityOutput{}, err
	}

	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return UpdateMasterNodeGroupActivityOutput{}, err
	}

	for _, np := range cluster.GetNodePools() {
		if input.NodePoolName == np.Name && np.Master && np.Count == 1 {
			return UpdateMasterNodeGroupActivityOutput{}, errors.Errorf("updating single master clusters is prohibited")
		}
	}

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return UpdateMasterNodeGroupActivityOutput{}, errors.Errorf("failed to cast to AWSCluster %t", cluster)
	}

	_, signedToken, err := a.tokenGenerator.GenerateClusterToken(input.OrganizationID, input.ClusterID)
	if err != nil {
		return UpdateMasterNodeGroupActivityOutput{}, errors.WrapIf(err, "failed to generate Pipeline token")
	}

	nodeLabels := []string{
		fmt.Sprintf("%v=%v", pkgCluster.NodePoolNameLabelKey, input.NodePoolName),
	}

	if input.NodePoolVersion != "" {
		nodeLabels = append(nodeLabels, fmt.Sprintf("%v=%v", pkgCluster.NodePoolVersionLabelKey, input.NodePoolVersion))
	}

	bootstrapCommand, err := awsCluster.GetBootstrapCommand(
		input.NodePoolName,
		a.externalBaseURL,
		a.externalBaseURLInsecure,
		signedToken,
		nodeLabels,
		input.Version,
	)
	if err != nil {
		return UpdateMasterNodeGroupActivityOutput{}, errors.WrapIf(err, "failed to fetch bootstrap command")
	}

	template, err := cloudformation2.GetCloudFormationTemplate(PKECloudFormationTemplateBasePath, "masters.cf.yaml")
	if err != nil {
		return UpdateMasterNodeGroupActivityOutput{}, errors.WrapIf(err, "loading CF template")
	}

	cloudformationClient := cloudformation.New(sess)

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:     aws.String("ClusterName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("InstanceType"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("VPCId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("VPCDefaultSecurityGroupId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:   aws.String("PkeCommand"),
			ParameterValue: aws.String(bootstrapCommand),
		},
		{
			ParameterKey:     aws.String("IamInstanceProfile"),
			UsePreviousValue: aws.Bool(true),
		},
		sdkCloudFormation.NewOptionalStackParameter(
			"ImageId",
			input.NodeImage != "",
			input.NodeImage,
		),
		sdkCloudFormation.NewOptionalStackParameter(
			"VolumeSize",
			input.NodeVolumeSize > 0,
			fmt.Sprint(input.NodeVolumeSize),
		),
		{
			ParameterKey:     aws.String("PkeVersion"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("KeyName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("TargetGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("SubnetIds"),
			UsePreviousValue: aws.Bool(true),
		},
	}

	// we don't reuse the creation time template, since it may have changed
	updateStackInput := &cloudformation.UpdateStackInput{
		ClientRequestToken: aws.String(sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID)),
		StackName:          aws.String(input.StackName),
		Capabilities:       aws.StringSlice([]string{cloudformation.CapabilityCapabilityAutoExpand}),
		Parameters:         stackParams,
		Tags:               getNodePoolStackTags(input.ClusterName, input.ClusterTags),
		TemplateBody:       aws.String(template),
	}

	_, err = cloudformationClient.UpdateStack(updateStackInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationError" && strings.HasPrefix(awsErr.Message(), awsNoUpdatesError) {
			return UpdateMasterNodeGroupActivityOutput{}, nil
		}

		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
				err = pkgCloudFormation.NewAwsStackFailure(err, input.StackName, aws.StringValue(updateStackInput.ClientRequestToken), cloudformationClient)
				err = errors.WrapIff(err, "waiting for %q CF stack update operation to complete failed", input.StackName)
				if pkgCloudFormation.IsErrorFinal(err) {
					return UpdateMasterNodeGroupActivityOutput{}, cadence.NewCustomError(ErrReasonStackFailed, err.Error())
				}
				return UpdateMasterNodeGroupActivityOutput{}, errors.WrapIff(err, "waiting for %q CF stack update operation to complete failed", input.StackName)
			}
		}
	}

	return UpdateMasterNodeGroupActivityOutput{NodePoolChanged: true}, nil
}
