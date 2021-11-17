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

	NodeVolumes    *eks.NodePoolVolumes
	NodeImage      string
	SecurityGroups []string

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

	effectiveSecurityGroups := input.SecurityGroups

	effectiveVolumes := input.NodeVolumes
	if effectiveVolumes == nil {
		effectiveVolumes = &eks.NodePoolVolumes{}
	}
	if effectiveVolumes.InstanceRoot == nil {
		effectiveVolumes.InstanceRoot = &eks.NodePoolVolume{}
	}
	if effectiveVolumes.KubeletRoot == nil {
		effectiveVolumes.KubeletRoot = &eks.NodePoolVolume{}
	}

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
			CustomNodeSecurityGroups           string         `mapstructure:"CustomNodeSecurityGroups,omitempty"` // Note: CustomNodeSecurityGroups is only available from template version 2.0.0.
			NodeImageID                        string         `mapstructure:"NodeImageId"`
			NodeVolumeStorage                  string         `mapstructure:"NodeVolumeStorage,omitempty"`           // Note: NodeVolumeStorage is only available from template version 2.5.0.
			NodeVolumeEncryptionEnabled        string         `mapstructure:"NodeVolumeEncryptionEnabled,omitempty"` // Note: NodeVolumeEncryptionEnabled is only available from template version 2.1.0.
			NodeVolumeEncryptionKeyARN         string         `mapstructure:"NodeVolumeEncryptionKeyARN,omitempty"`  // Note: NodeVolumeEncryptionKeyARN is only available from template version 2.1.0.
			NodeVolumeSize                     int            `mapstructure:"NodeVolumeSize"`
			KubeletRootVolumeStorage           string         `mapstructure:"KubeletRootVolumeStorage,omitempty"`           // Note: KubeletRootVolumeStorage is only available from template version 2.5.0.
			KubeletRootVolumeEncryptionEnabled string         `mapstructure:"KubeletRootVolumeEncryptionEnabled,omitempty"` // Note: KubeletRootVolumeEncryptionEnabled is only available from template version 2.5.0.
			KubeletRootVolumeEncryptionKeyARN  string         `mapstructure:"KubeletRootVolumeEncryptionKeyARN,omitempty"`  // Note: KubeletRootVolumeEncryptionKeyARN is only available from template version 2.5.0.
			KubeletRootVolumeSize              int            `mapstructure:"KubeletRootVolumeSize,omitempty"`              // Note: KubeletRootVolumeSize is only available from template version 2.5.0.
			TemplateVersion                    semver.Version `mapstructure:"TemplateVersion,omitempty"`                    // Note: TemplateVersion is only available from template version 2.0.0.
		}
		err = sdkCloudFormation.ParseStackParameters(getCFStackOutput.Stack.Parameters, &parameters)
		if err != nil {
			return err
		}

		currentTemplateVersion = parameters.TemplateVersion

		if effectiveImage == "" {
			effectiveImage = parameters.NodeImageID
		}

		if effectiveVolumes.InstanceRoot.Storage == "" {
			effectiveVolumes.InstanceRoot.Storage = parameters.NodeVolumeStorage
			// set default ebs value for InstanceRoot.Storage for old templates
			if currentTemplateVersion.IsLessThan("2.5.0") {
				effectiveVolumes.InstanceRoot.Storage = eks.EBS_STORAGE
			}
		}
		// load EBS volume related params only in case storage == ebs
		if eks.EBS_STORAGE == effectiveVolumes.InstanceRoot.Storage {
			if effectiveVolumes.InstanceRoot.Encryption == nil &&
				parameters.NodeVolumeEncryptionEnabled != "" {
				isNodeVolumeEncryptionEnabled, err := strconv.ParseBool(parameters.NodeVolumeEncryptionEnabled)
				if err != nil {
					return errors.WrapIf(err, "invalid node volume encryption enabled value")
				}

				effectiveVolumes.InstanceRoot.Encryption = &eks.NodePoolVolumeEncryption{
					Enabled:          isNodeVolumeEncryptionEnabled,
					EncryptionKeyARN: parameters.NodeVolumeEncryptionKeyARN,
				}
			}

			if effectiveVolumes.InstanceRoot.Size == 0 {
				effectiveVolumes.InstanceRoot.Size = parameters.NodeVolumeSize
			}
			if effectiveVolumes.InstanceRoot.Type == "" {
				effectiveVolumes.InstanceRoot.Type = "gp3"
			}
		} else if eks.INSTANCE_STORE_STORAGE == effectiveVolumes.InstanceRoot.Storage {
			effectiveVolumes.InstanceRoot.Encryption = nil
			// can not be set to empty string
			effectiveVolumes.InstanceRoot.Type = "gp3"
			effectiveVolumes.InstanceRoot.Size = 0
		}

		if effectiveVolumes.KubeletRoot.Storage == "" {
			// set default none value for KubeletRoot.Storage for old templates
			if currentTemplateVersion.IsLessThan("2.5.0") {
				effectiveVolumes.KubeletRoot.Storage = eks.NONE_STORAGE
			} else {
				effectiveVolumes.KubeletRoot.Storage = parameters.KubeletRootVolumeStorage
			}
		}

		// load EBS volume related params only in case storage == ebs
		if eks.EBS_STORAGE == effectiveVolumes.KubeletRoot.Storage {
			if effectiveVolumes.KubeletRoot.Encryption == nil &&
				parameters.KubeletRootVolumeEncryptionEnabled != "" {
				isVolumeEncryptionEnabled, err := strconv.ParseBool(parameters.KubeletRootVolumeEncryptionEnabled)
				if err != nil {
					return errors.WrapIf(err, "invalid kubelet root volume encryption enabled value")
				}

				effectiveVolumes.KubeletRoot.Encryption = &eks.NodePoolVolumeEncryption{
					Enabled:          isVolumeEncryptionEnabled,
					EncryptionKeyARN: parameters.KubeletRootVolumeEncryptionKeyARN,
				}
			}

			if effectiveVolumes.KubeletRoot.Size == 0 {
				effectiveVolumes.KubeletRoot.Size = parameters.KubeletRootVolumeSize
			}
			// set default size in case it's still 0
			if effectiveVolumes.KubeletRoot.Size == 0 {
				effectiveVolumes.KubeletRoot.Size = 50
			}
			if effectiveVolumes.KubeletRoot.Type == "" {
				effectiveVolumes.KubeletRoot.Type = "gp3"
			}
		} else if eks.INSTANCE_STORE_STORAGE == effectiveVolumes.KubeletRoot.Storage ||
			eks.NONE_STORAGE == effectiveVolumes.KubeletRoot.Storage {
			effectiveVolumes.KubeletRoot.Encryption = nil
			effectiveVolumes.KubeletRoot.Type = ""
			effectiveVolumes.KubeletRoot.Size = 0
		}

		if effectiveSecurityGroups == nil &&
			parameters.CustomNodeSecurityGroups != "" {
			effectiveSecurityGroups = strings.Split(parameters.CustomNodeSecurityGroups, ",")
			sort.Strings(effectiveSecurityGroups)
		}
	}

	var volumeSize int
	if input.NodeVolumes != nil && input.NodeVolumes.InstanceRoot != nil && input.NodeVolumes.InstanceRoot.Size > 0 {
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
				OptionalVolumeSize: input.NodeVolumes.InstanceRoot.Size,
			}
			var activityOutput eksWorkflow.SelectVolumeSizeActivityOutput
			processActivity := process.StartActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName)
			err = workflow.ExecuteActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
			processActivity.Finish(ctx, err)
			if err != nil {
				return err
			}

			volumeSize = activityOutput.VolumeSize
			effectiveVolumes.InstanceRoot.Size = volumeSize
		}
	}

	var nodePoolVersion string
	{
		activityInput := eksWorkflow.CalculateNodePoolVersionActivityInput{
			Image:                effectiveImage,
			Volumes:              *effectiveVolumes,
			CustomSecurityGroups: effectiveSecurityGroups,
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
			NodeVolumes:            *effectiveVolumes,
			NodeImage:              input.NodeImage,
			SecurityGroups:         input.SecurityGroups,
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
