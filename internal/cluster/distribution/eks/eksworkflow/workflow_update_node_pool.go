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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	"github.com/banzaicloud/pipeline/pkg/sdk/cadence/lib/pipeline/processlog"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/pkg/sdk/semver"
)

const UpdateNodePoolWorkflowName = "eks-update-node-pool"

type UpdateNodePoolWorkflow struct {
	awsFactory            awsworkflow.AWSFactory
	cloudFormationFactory awsworkflow.CloudFormationAPIFactory
	processLogger         processlog.ProcessLogger
}

// NewUpdateNodePoolWorkflow returns a new UpdateNodePoolWorkflow.
func NewUpdateNodePoolWorkflow(
	awsFactory awsworkflow.AWSFactory,
	cloudFormationFactory awsworkflow.CloudFormationAPIFactory,
	processLogger processlog.ProcessLogger,
) UpdateNodePoolWorkflow {
	return UpdateNodePoolWorkflow{
		awsFactory:            awsFactory,
		cloudFormationFactory: cloudFormationFactory,
		processLogger:         processLogger,
	}
}

type UpdateNodePoolWorkflowInput struct {
	ProviderSecretID string
	Region           string

	StackName string

	OrganizationID  uint
	ClusterID       uint
	ClusterSecretID string
	ClusterName     string
	NodePoolName    string

	NodeVolumeEncryption *eks.NodePoolVolumeEncryption
	NodeVolumeSize       int
	NodeImage            string
	SecurityGroups       []string
	UseInstanceStore     *bool

	Options eks.NodePoolUpdateOptions

	ClusterTags map[string]string
}

func (w UpdateNodePoolWorkflow) Register(worker worker.Registry) {
	worker.RegisterWorkflowWithOptions(w.Execute, workflow.RegisterOptions{Name: UpdateNodePoolWorkflowName})
}

func (w UpdateNodePoolWorkflow) Execute(ctx workflow.Context, input UpdateNodePoolWorkflowInput) (err error) {
	activityOptions := workflow.ActivityOptions{
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 1.01,
			MaximumAttempts:    10,
			MaximumInterval:    10 * time.Minute,
		},
		ScheduleToStartTimeout: time.Duration(workflow.GetInfo(ctx).ExecutionStartToCloseTimeoutSeconds) * time.Second,
		StartToCloseTimeout:    5 * time.Minute,
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	clusterID := brn.New(input.OrganizationID, brn.ClusterResourceType, fmt.Sprint(input.ClusterID))

	process := w.processLogger.StartProcess(ctx, clusterID.String())
	defer func() {
		process.Finish(ctx, err)
	}()
	defer func() {
		status := cluster.Running
		statusMessage := cluster.RunningMessage

		if err != nil {
			if cadence.IsCanceledError(err) {
				ctx, _ = workflow.NewDisconnectedContext(ctx)
			}

			status = cluster.Warning
			statusMessage = fmt.Sprintf("failed to update node pool: %s", err.Error())
		}

		_ = setClusterStatus(ctx, input.ClusterID, status, statusMessage)
	}()

	providerSecretID, err := brn.Parse(input.ProviderSecretID)
	if err != nil {
		return err
	}

	eksActivityInput := eksWorkflow.EKSActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       providerSecretID.ResourceID,
		Region:         input.Region,
		ClusterName:    input.ClusterName,
	}

	var currentTemplateVersion semver.Version
	effectiveImage := input.NodeImage
	effectiveVolumeEncryption := input.NodeVolumeEncryption
	effectiveVolumeSize := input.NodeVolumeSize
	effectiveSecurityGroups := input.SecurityGroups
	effectiveUseInstanceStore := input.UseInstanceStore

	{ // Note: needing CF stack for template version and possibly node pool version.
		getCFStackInput := eksWorkflow.GetCFStackActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, input.NodePoolName),
		}
		var getCFStackOutput eksWorkflow.GetCFStackActivityOutput
		processActivity := process.StartActivity(ctx, eksWorkflow.GetCFStackActivityName)
		err = workflow.ExecuteActivity(ctx, eksWorkflow.GetCFStackActivityName, getCFStackInput).Get(ctx, &getCFStackOutput)
		processActivity.Finish(ctx, err)
		if err != nil {
			return err
		}

		var parameters struct {
			CustomNodeSecurityGroups    string         `mapstructure:"CustomNodeSecurityGroups,omitempty"` // Note: CustomNodeSecurityGroups is only available from template version 2.0.0.
			NodeImageID                 string         `mapstructure:"NodeImageId"`
			NodeVolumeEncryptionEnabled string         `mapstructure:"NodeVolumeEncryptionEnabled,omitempty"` // Note: NodeVolumeEncryptionEnabled is only available from template version 2.1.0.
			NodeVolumeEncryptionKeyARN  string         `mapstructure:"NodeVolumeEncryptionKeyARN,omitempty"`  // Note: NodeVolumeEncryptionKeyARN is only available from template version 2.1.0.
			NodeVolumeSize              int            `mapstructure:"NodeVolumeSize"`
			TemplateVersion             semver.Version `mapstructure:"TemplateVersion,omitempty"`  // Note: TemplateVersion is only available from template version 2.0.0.
			UseInstanceStore            string         `mapstructure:"UseInstanceStore,omitempty"` // Note: TemplateVersion is only available from template version 2.2.0.

		}
		err = sdkCloudFormation.ParseStackParameters(getCFStackOutput.Stack.Parameters, &parameters)
		if err != nil {
			return err
		}

		currentTemplateVersion = parameters.TemplateVersion

		if effectiveImage == "" {
			effectiveImage = parameters.NodeImageID
		}

		if effectiveVolumeEncryption == nil &&
			parameters.NodeVolumeEncryptionEnabled != "" {
			isNodeVolumeEncryptionEnabled, err := strconv.ParseBool(parameters.NodeVolumeEncryptionEnabled)
			if err != nil {
				return errors.WrapIf(err, "invalid node volume encryption enabled value")
			}

			effectiveVolumeEncryption = &eks.NodePoolVolumeEncryption{
				Enabled:          isNodeVolumeEncryptionEnabled,
				EncryptionKeyARN: parameters.NodeVolumeEncryptionKeyARN,
			}
		}

		if effectiveVolumeSize == 0 {
			effectiveVolumeSize = parameters.NodeVolumeSize
		}

		if effectiveSecurityGroups == nil &&
			parameters.CustomNodeSecurityGroups != "" {
			effectiveSecurityGroups = strings.Split(parameters.CustomNodeSecurityGroups, ",")
			sort.Strings(effectiveSecurityGroups)
		}

		if effectiveUseInstanceStore == nil &&
			parameters.UseInstanceStore != "" {
			useInstanceStore, err := strconv.ParseBool(parameters.UseInstanceStore)
			if err != nil {
				return errors.WrapIf(err, "invalid UseInstanceStore parameter value")
			}
			effectiveUseInstanceStore = &useInstanceStore
		}
	}

	var volumeSize int
	if input.NodeVolumeSize > 0 {
		var amiSize int
		{
			activityInput := eksWorkflow.GetAMISizeActivityInput{
				EKSActivityInput: eksActivityInput,
				ImageID:          effectiveImage,
			}
			var activityOutput eksWorkflow.GetAMISizeActivityOutput
			processActivity := process.StartActivity(ctx, eksWorkflow.GetAMISizeActivityName)
			err = workflow.ExecuteActivity(ctx, eksWorkflow.GetAMISizeActivityName, activityInput).Get(ctx, &activityOutput)
			processActivity.Finish(ctx, err)
			if err != nil {
				return err
			}

			amiSize = activityOutput.AMISize
		}

		{
			activityInput := eksWorkflow.SelectVolumeSizeActivityInput{
				AMISize:            amiSize,
				OptionalVolumeSize: input.NodeVolumeSize,
			}
			var activityOutput eksWorkflow.SelectVolumeSizeActivityOutput
			processActivity := process.StartActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName)
			err = workflow.ExecuteActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
			processActivity.Finish(ctx, err)
			if err != nil {
				return err
			}

			volumeSize = activityOutput.VolumeSize
			effectiveVolumeSize = volumeSize
		}
	}

	var nodePoolVersion string
	{
		activityInput := eksWorkflow.CalculateNodePoolVersionActivityInput{
			Image:                effectiveImage,
			VolumeEncryption:     effectiveVolumeEncryption,
			VolumeSize:           effectiveVolumeSize,
			CustomSecurityGroups: effectiveSecurityGroups,
			UseInstanceStore:     effectiveUseInstanceStore,
		}

		var output eksWorkflow.CalculateNodePoolVersionActivityOutput

		processActivity := process.StartActivity(ctx, eksWorkflow.CalculateNodePoolVersionActivityName)
		err = workflow.ExecuteActivity(
			ctx, eksWorkflow.CalculateNodePoolVersionActivityName, activityInput).Get(ctx, &output)
		processActivity.Finish(ctx, err)
		if err != nil {
			return err
		}

		nodePoolVersion = output.Version
	}

	{
		activityInput := UpdateNodeGroupActivityInput{
			SecretID:               input.ProviderSecretID,
			Region:                 input.Region,
			ClusterName:            input.ClusterName,
			StackName:              input.StackName,
			NodePoolName:           input.NodePoolName,
			NodePoolVersion:        nodePoolVersion,
			NodeVolumeEncryption:   input.NodeVolumeEncryption,
			NodeVolumeSize:         volumeSize,
			NodeImage:              input.NodeImage,
			SecurityGroups:         input.SecurityGroups,
			UseInstanceStore:       input.UseInstanceStore,
			MaxBatchSize:           input.Options.MaxBatchSize,
			MinInstancesInService:  input.Options.MaxSurge,
			ClusterTags:            input.ClusterTags,
			CurrentTemplateVersion: currentTemplateVersion,
		}

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 5 * time.Minute
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:          20 * time.Second,
			BackoffCoefficient:       1.1,
			MaximumAttempts:          10,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		}

		var output UpdateNodeGroupActivityOutput

		processActivity := process.StartActivity(ctx, UpdateNodeGroupActivityName)
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			UpdateNodeGroupActivityName,
			activityInput,
		).Get(ctx, &output)
		processActivity.Finish(ctx, err)
		if err != nil || !output.NodePoolChanged {
			return
		}
	}

	{
		activityInput := WaitCloudFormationStackUpdateActivityInput{
			SecretID:  input.ProviderSecretID,
			Region:    input.Region,
			StackName: input.StackName,
		}

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 100 * 10 * time.Minute // TODO: calculate based on desired node count (limited to around 100 nodes now)
		activityOptions.HeartbeatTimeout = time.Minute
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:          20 * time.Second,
			BackoffCoefficient:       1.1,
			MaximumAttempts:          20,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		}

		processActivity := process.StartActivity(ctx, WaitCloudFormationStackUpdateActivityName)
		err = pkgCadence.UnwrapError(workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			WaitCloudFormationStackUpdateActivityName,
			activityInput,
		).Get(ctx, nil))
		processActivity.Finish(ctx, err)
		if err != nil {
			return err
		}
	}

	return nil
}
