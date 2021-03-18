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

package eksworkflow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	pkgCloudFormation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/pkg/sdk/semver"
)

const awsNoUpdatesError = "No updates are to be performed."

const UpdateNodeGroupActivityName = "eks-update-node-group"

// UpdateNodeGroupActivity updates an existing node group.
type UpdateNodeGroupActivity struct {
	sessionFactory AWSSessionFactory

	// body of the cloud formation template
	cloudFormationTemplate      string
	defaultNodeVolumeEncryption *eks.NodePoolVolumeEncryption
}

// UpdateNodeGroupActivityInput holds the parameters for the node group update.
type UpdateNodeGroupActivityInput struct {
	SecretID string
	Region   string

	ClusterName string

	StackName string

	NodePoolName    string
	NodePoolVersion string

	NodeVolumeEncryption *eks.NodePoolVolumeEncryption
	NodeVolumeSize       int
	NodeImage            string
	DesiredCapacity      int64
	SecurityGroups       []string
	UseInstanceStore     *bool

	MaxBatchSize          int
	MinInstancesInService int

	ClusterTags map[string]string

	CurrentTemplateVersion semver.Version
}

type UpdateNodeGroupActivityOutput struct {
	NodePoolChanged bool
}

// NewUpdateNodeGroupActivity creates a new UpdateNodeGroupActivity instance.
func NewUpdateNodeGroupActivity(
	sessionFactory AWSSessionFactory,
	cloudFormationTemplate string,
	defaultNodeVolumeEncryption *eks.NodePoolVolumeEncryption,
) UpdateNodeGroupActivity {
	return UpdateNodeGroupActivity{
		sessionFactory:              sessionFactory,
		cloudFormationTemplate:      cloudFormationTemplate,
		defaultNodeVolumeEncryption: defaultNodeVolumeEncryption,
	}
}

// Register registers the activity in the worker.
func (a UpdateNodeGroupActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: UpdateNodeGroupActivityName})
}

// Execute is the main body of the activity, returns true if there was any update and that was successful.
func (a UpdateNodeGroupActivity) Execute(ctx context.Context, input UpdateNodeGroupActivityInput) (UpdateNodeGroupActivityOutput, error) {
	sess, err := a.sessionFactory.NewSession(input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil { // internal error?
		return UpdateNodeGroupActivityOutput{}, err
	}

	cloudformationClient := cloudformation.New(sess)

	nodeLabels := []string{
		fmt.Sprintf("%v=%v", cluster.NodePoolNameLabelKey, input.NodePoolName),
	}

	if input.NodePoolVersion != "" {
		nodeLabels = append(nodeLabels, fmt.Sprintf("%v=%v", cluster.NodePoolVersionLabelKey, input.NodePoolVersion))
	}

	nodeVolumeEncryptionEnabled := ""
	if input.NodeVolumeEncryption != nil {
		nodeVolumeEncryptionEnabled = strconv.FormatBool(input.NodeVolumeEncryption.Enabled)
	} else if input.CurrentTemplateVersion.IsLessThan("2.1.0") &&
		a.defaultNodeVolumeEncryption != nil { // Note: old stack, Pipeline default should take precedence over AWS default.
		nodeVolumeEncryptionEnabled = strconv.FormatBool(a.defaultNodeVolumeEncryption.Enabled)
	}

	nodeVolumeEncryptionKeyARN := ""
	if nodeVolumeEncryptionEnabled == "true" &&
		input.NodeVolumeEncryption != nil &&
		input.NodeVolumeEncryption.EncryptionKeyARN != "" {
		nodeVolumeEncryptionKeyARN = input.NodeVolumeEncryption.EncryptionKeyARN
	} else if nodeVolumeEncryptionEnabled == "true" &&
		a.defaultNodeVolumeEncryption != nil &&
		a.defaultNodeVolumeEncryption.EncryptionKeyARN != "" {
		nodeVolumeEncryptionKeyARN = a.defaultNodeVolumeEncryption.EncryptionKeyARN
	} else if input.CurrentTemplateVersion.IsLessThan("2.1.0") &&
		a.defaultNodeVolumeEncryption != nil { // Note: old stack, Pipeline default should take precedence over AWS default.
		nodeVolumeEncryptionKeyARN = a.defaultNodeVolumeEncryption.EncryptionKeyARN
	}

	tags := getNodePoolStackTags(input.ClusterName, input.ClusterTags)

	var stackTagsBuilder strings.Builder
	for tagIndex, tag := range tags {
		if tagIndex != 0 {
			_, _ = stackTagsBuilder.WriteString(",")
		}

		_, _ = stackTagsBuilder.WriteString(aws.StringValue(tag.Key) + "=" + aws.StringValue(tag.Value))
	}

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:     aws.String("KeyName"),
			UsePreviousValue: aws.Bool(true),
		},
		sdkCloudFormation.NewOptionalStackParameter(
			"NodeImageId",
			input.NodeImage != "",
			input.NodeImage,
		),
		sdkCloudFormation.NewOptionalStackParameter(
			"CustomNodeSecurityGroups",
			input.SecurityGroups != nil || input.CurrentTemplateVersion.IsLessThan("2.0.0"), // Note: older templates cannot use non-existing previous value.
			strings.Join(input.SecurityGroups, ","),
		),
		{
			ParameterKey:     aws.String("NodeInstanceType"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeSpotPrice"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeAutoScalingGroupMinSize"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeAutoScalingGroupMaxSize"),
			UsePreviousValue: aws.Bool(true),
		},
		sdkCloudFormation.NewOptionalStackParameter(
			"NodeAutoScalingGroupMaxBatchSize",
			input.MaxBatchSize > 0,
			fmt.Sprintf("%d", input.MaxBatchSize),
		),
		{
			ParameterKey:   aws.String("NodeAutoScalingGroupMinInstancesInService"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.MinInstancesInService)),
		},
		sdkCloudFormation.NewOptionalStackParameter(
			"NodeAutoScalingInitSize",
			input.DesiredCapacity > 0,
			fmt.Sprintf("%d", input.DesiredCapacity),
		),
		sdkCloudFormation.NewOptionalStackParameter(
			"NodeVolumeEncryptionEnabled",
			nodeVolumeEncryptionEnabled != "" || input.CurrentTemplateVersion.IsLessThan("2.1.0"), // Note: older templates cannot use non-existing previous value.
			nodeVolumeEncryptionEnabled,
		),
		sdkCloudFormation.NewOptionalStackParameter(
			"NodeVolumeEncryptionKeyARN",
			nodeVolumeEncryptionEnabled != "" || input.CurrentTemplateVersion.IsLessThan("2.1.0"), // Note: when enablement is set, key ARN should be updated. // Note: older templates cannot use non-existing previous value.
			nodeVolumeEncryptionKeyARN,
		),
		sdkCloudFormation.NewOptionalStackParameter(
			"NodeVolumeSize",
			input.NodeVolumeSize > 0,
			fmt.Sprintf("%d", input.NodeVolumeSize),
		),
		{
			ParameterKey:   aws.String("StackTags"),
			ParameterValue: aws.String(stackTagsBuilder.String()),
		},
		{
			ParameterKey:     aws.String("ClusterName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeGroupName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("ClusterControlPlaneSecurityGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeSecurityGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("VpcId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("Subnets"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeInstanceRoleId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("ClusterAutoscalerEnabled"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("TerminationDetachEnabled"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:   aws.String("KubeletExtraArguments"),
			ParameterValue: aws.String(fmt.Sprintf("--node-labels %v", strings.Join(nodeLabels, ","))),
		},
		sdkCloudFormation.NewOptionalStackParameter(
			"UseInstanceStore",
			input.UseInstanceStore != nil || input.CurrentTemplateVersion.IsLessThan("2.2.0"),
			strconv.FormatBool(aws.BoolValue(input.UseInstanceStore)), // Note: false default value for old stacks.
		),
	}

	// we don't reuse the creation time template, since it may have changed
	updateStackInput := &cloudformation.UpdateStackInput{
		ClientRequestToken: aws.String(sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID)),
		StackName:          aws.String(input.StackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:         stackParams,
		Tags:               tags,
		TemplateBody:       aws.String(a.cloudFormationTemplate),
	}

	_, err = cloudformationClient.UpdateStack(updateStackInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationError" && strings.HasPrefix(awsErr.Message(), awsNoUpdatesError) {
			return UpdateNodeGroupActivityOutput{}, nil
		}

		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
				err = pkgCloudFormation.NewAwsStackFailure(err, input.StackName, aws.StringValue(updateStackInput.ClientRequestToken), cloudformationClient)
				err = errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
				if pkgCloudFormation.IsErrorFinal(err) {
					return UpdateNodeGroupActivityOutput{}, cadence.NewCustomError(ErrReasonStackFailed, err.Error())
				}
				return UpdateNodeGroupActivityOutput{}, errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
			}
		}

		return UpdateNodeGroupActivityOutput{}, err
	}

	return UpdateNodeGroupActivityOutput{NodePoolChanged: true}, nil
}
