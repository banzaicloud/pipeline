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

package workflow

import (
	"fmt"
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/cluster"
)

const CreateClusterWorkflowName = "pke-azure-create-cluster"

// CreateClusterWorkflowInput
type CreateClusterWorkflowInput struct {
	ClusterID                       uint
	ClusterName                     string
	ClusterUID                      string
	OrganizationID                  uint
	OrganizationName                string
	ResourceGroupName               string
	SecretID                        string
	Distribution                    string
	OIDCEnabled                     bool
	VirtualNetworkTemplate          VirtualNetworkTemplate
	LoadBalancerTemplates           []LoadBalancerTemplate
	PublicIPAddress                 PublicIPAddress
	RoleAssignmentTemplates         []RoleAssignmentTemplate
	RouteTable                      RouteTable
	SecurityGroups                  []SecurityGroup
	VirtualMachineScaleSetTemplates []VirtualMachineScaleSetTemplate
	NodePoolLabels                  map[string]map[string]string
	HTTPProxy                       intPKE.HTTPProxy
	AccessPoints                    pke.AccessPoints
	APIServerAccessPoints           pke.APIServerAccessPoints
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		WaitForCancellation:    true,
	}
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 1 * time.Hour,
		TaskStartToCloseTimeout:      30 * time.Second,
	}
	ctx = workflow.WithChildOptions(workflow.WithActivityOptions(ctx, ao), cwo)

	// Generate CA certificates
	{
		activityInput := pkeworkflow.GenerateCertificatesActivityInput{ClusterID: input.ClusterID}

		err := workflow.ExecuteActivity(ctx, pkeworkflow.GenerateCertificatesActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	// Create dex client for the cluster
	if input.OIDCEnabled {
		activityInput := pkeworkflow.CreateDexClientActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, pkeworkflow.CreateDexClientActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	infraInput := CreateAzureInfrastructureWorkflowInput{
		OrganizationID:        input.OrganizationID,
		ClusterID:             input.ClusterID,
		ClusterName:           input.ClusterName,
		SecretID:              input.SecretID,
		ResourceGroupName:     input.ResourceGroupName,
		LoadBalancers:         input.LoadBalancerTemplates,
		PublicIPAddress:       input.PublicIPAddress,
		RoleAssignments:       input.RoleAssignmentTemplates,
		RouteTable:            input.RouteTable,
		ScaleSets:             input.VirtualMachineScaleSetTemplates,
		SecurityGroups:        input.SecurityGroups,
		VirtualNetwork:        input.VirtualNetworkTemplate,
		HTTPProxy:             input.HTTPProxy,
		AccessPoints:          input.AccessPoints,
		APIServerAccessPoints: input.APIServerAccessPoints,
	}
	err := workflow.ExecuteChildWorkflow(ctx, CreateInfraWorkflowName, infraInput).Get(ctx, nil)
	if err != nil {
		_ = setClusterErrorStatus(ctx, input.ClusterID, err)
		return err
	}

	setClusterStatus(ctx, input.ClusterID, pkgCluster.Creating, "waiting for Kubernetes master") // nolint: errcheck

	if err = waitForMasterReadySignal(ctx, 1*time.Hour); err != nil {
		_ = setClusterErrorStatus(ctx, input.ClusterID, err)
		return err
	}

	var configSecretID string
	{
		activityInput := cluster.DownloadK8sConfigActivityInput{
			ClusterID: input.ClusterID,
		}
		future := workflow.ExecuteActivity(ctx, cluster.DownloadK8sConfigActivityName, activityInput)
		if err := future.Get(ctx, &configSecretID); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	{
		workflowInput := clustersetup.WorkflowInput{
			ConfigSecretID: brn.New(input.OrganizationID, brn.SecretResourceType, configSecretID).String(),
			Cluster: clustersetup.Cluster{
				ID:           input.ClusterID,
				UID:          input.ClusterUID,
				Name:         input.ClusterName,
				Distribution: input.Distribution,
			},
			Organization: clustersetup.Organization{
				ID:   input.OrganizationID,
				Name: input.OrganizationName,
			},
			NodePoolLabels: input.NodePoolLabels,
		}

		future := workflow.ExecuteChildWorkflow(ctx, clustersetup.WorkflowName, workflowInput)
		if err := future.Get(ctx, nil); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	postHookWorkflowInput := cluster.RunPostHooksWorkflowInput{
		ClusterID: input.ClusterID,
		PostHooks: cluster.BuildWorkflowPostHookFunctions(nil, true),
	}

	err = workflow.ExecuteChildWorkflow(ctx, cluster.RunPostHooksWorkflowName, postHookWorkflowInput).Get(ctx, nil)
	if err != nil {
		_ = setClusterErrorStatus(ctx, input.ClusterID, err)
		return err
	}

	return nil
}

func waitForMasterReadySignal(ctx workflow.Context, timeout time.Duration) error {
	signalName := "master-ready"
	signalChan := workflow.GetSignalChannel(ctx, signalName)
	signalTimeoutTimer := workflow.NewTimer(ctx, timeout)
	signalTimeout := false

	signalSelector := workflow.NewSelector(ctx).AddReceive(signalChan, func(c workflow.Channel, more bool) {
		c.Receive(ctx, nil)
		workflow.GetLogger(ctx).Info("Received signal!", zap.String("signal", signalName))
	}).AddFuture(signalTimeoutTimer, func(workflow.Future) {
		signalTimeout = true
	})

	signalSelector.Select(ctx) // wait for signal

	if signalTimeout {
		return fmt.Errorf("timeout while waiting for %q signal", signalName)
	}
	return nil
}
