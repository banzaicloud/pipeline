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

package clustersetup

import (
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/integratedservices/operator"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// WorkflowName can be used to reference the cluster setup workflow.
const WorkflowName = "cluster-setup"

const (
	DeployClusterAutoscalerActivityName = "deploy-cluster-autoscaler"
	RestoreBackupActivityName           = "restore-backup"
	ErrReasonRestoreFailed              = "BACKUP_RESTORE_FAILED"
)

// Workflow orchestrates the post-creation cluster setup flow.
type Workflow struct {
	// InstallInit
	InstallInitManifest bool

	// Drives installation
	IsIntegratedServicesV2 bool

	// Install additional Pipeline components here
	PipelineNamespace string

	InstallHelmCharts []HelmChartInstallParams
}

type HelmChartInstallParams struct {
	ReleaseName  string
	ChartName    string
	ChartVersion string
	Values       []byte
}

// WorkflowInput is the input for a cluster setup workflow.
type WorkflowInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string

	// Cluster information
	Cluster      Cluster
	Organization Organization

	NodePoolLabels      map[string]map[string]string
	RestoreBackupParams *pkgCluster.RestoreFromBackupParams
}

type DeployClusterAutoscalerActivityInput struct {
	ClusterID uint
}

type RestoreBackupActivityInput struct {
	ClusterID           uint
	RestoreBackupParams pkgCluster.RestoreFromBackupParams
}

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	ID           uint
	UID          string
	Name         string
	Distribution string
	Cloud        string
}

// Organization contains information about the organization a cluster belongs to.
type Organization struct {
	ID   uint
	Name string
}

// Execute executes the cluster setup workflow.
func (w Workflow) Execute(ctx workflow.Context, input WorkflowInput) error {
	// Default timeouts and retries
	activityOptions := workflow.ActivityOptions{
		ScheduleToStartTimeout: 20 * time.Minute,
		StartToCloseTimeout:    30 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          30,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonRestoreFailed},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Install the cluster manifest to the cluster (if configured)
	if w.InstallInitManifest {
		activityInput := InitManifestActivityInput{
			ConfigSecretID: input.ConfigSecretID,
			Cluster:        input.Cluster,
			Organization:   input.Organization,
		}

		err := workflow.ExecuteActivity(ctx, InitManifestActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := CreatePipelineNamespaceActivityInput{
			ConfigSecretID: input.ConfigSecretID,
		}

		err := workflow.ExecuteActivity(ctx, CreatePipelineNamespaceActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := LabelKubeSystemNamespaceActivityInput{
			ConfigSecretID: input.ConfigSecretID,
		}

		err := workflow.ExecuteActivity(ctx, LabelKubeSystemNamespaceActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := InstallNodePoolLabelSetOperatorActivityInput{
			ClusterID: input.Cluster.ID,
		}

		err := workflow.ExecuteActivity(ctx, InstallNodePoolLabelSetOperatorActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: input.ConfigSecretID,
			Labels:         input.NodePoolLabels,
		}

		err := workflow.ExecuteActivity(ctx, ConfigureNodePoolLabelsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := DeployClusterAutoscalerActivityInput{
			ClusterID: input.Cluster.ID,
		}

		err := workflow.ExecuteActivity(ctx, DeployClusterAutoscalerActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		activityInput := DeployIngressControllerActivityInput{
			ClusterID: input.Cluster.ID,
			OrgID:     input.Organization.ID,
			Cloud:     input.Cluster.Cloud,
		}

		err := workflow.ExecuteActivity(ctx, DeployIngressControllerActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}
	{
		activityInput := DeployInstanceTerminationHandlerActivityInput{
			ClusterID:   input.Cluster.ID,
			OrgID:       input.Organization.ID,
			Cloud:       input.Cluster.Cloud,
			ClusterName: input.Cluster.Name,
		}

		err := workflow.ExecuteActivity(ctx, DeployInstanceTerminationHandlerActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	{
		if w.IsIntegratedServicesV2 {
			// install / upgrade the  integrated service operator
			input := operator.NewInstallerActivityInput(input.Organization.ID, input.Cluster.ID)
			if err := workflow.ExecuteActivity(ctx, operator.IntegratedServiceOperatorInstallerActivityName, input).Get(ctx, nil); err != nil {
				return errors.WrapIfWithDetails(err, "failed to install the  operator", "orgID", input.OrgID, "clusterID", input.ClusterID)
			}
		}
	}

	{
		for _, chart := range w.InstallHelmCharts {
			input := HelmInstallActivityInput{
				ClusterID:    input.Cluster.ID,
				Namespace:    w.PipelineNamespace,
				ReleaseName:  chart.ReleaseName,
				ChartName:    chart.ChartName,
				ChartVersion: chart.ChartVersion,
				Values:       chart.Values,
			}

			if err := workflow.ExecuteActivity(ctx, HelmInstallActivityName, input).Get(ctx, nil); err != nil {
				return errors.WrapIfWithDetails(err, "cluster setup failed", "clusterID", input.ClusterID)
			}
		}
	}

	if input.RestoreBackupParams != nil {
		activityInput := RestoreBackupActivityInput{
			ClusterID:           input.Cluster.ID,
			RestoreBackupParams: *input.RestoreBackupParams,
		}

		err := workflow.ExecuteActivity(ctx, RestoreBackupActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
