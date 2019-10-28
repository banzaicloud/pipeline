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

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/cluster"
)

const CreateClusterWorkflowName = "pke-vsphere-create-cluster"

// CreateClusterWorkflowInput
type CreateClusterWorkflowInput struct {
	ClusterID        uint
	ClusterName      string
	ClusterUID       string
	OrganizationID   uint
	OrganizationName string
	ResourcePoolName string
	FolderName       string
	DatastoreName    string
	SecretID         string
	OIDCEnabled      bool
	PostHooks        pkgCluster.PostHooks
	Nodes            []Node
	HTTPProxy        intPKE.HTTPProxy
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Generate CA certificates
	{
		activityInput := pkeworkflow.GenerateCertificatesActivityInput{ClusterID: input.ClusterID}

		err := workflow.ExecuteActivity(ctx, pkeworkflow.GenerateCertificatesActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			// TODO _ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	/*
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
	*/

	// Create nodes
	{
		futures := make([]workflow.Future, len(input.Nodes))

		for i, node := range input.Nodes {
			if node.UserDataScriptParams == nil {
				node.UserDataScriptParams = make(map[string]string)
			}
			activityInput := CreateNodeActivityInput{
				OrganizationID:   input.OrganizationID,
				SecretID:         input.SecretID,
				ClusterID:        input.ClusterID,
				ClusterName:      input.ClusterName,
				ResourcePoolName: input.ResourcePoolName,
				FolderName:       input.FolderName,
				DatastoreName:    input.DatastoreName,
				Node:             node,
			}
			futures[i] = workflow.ExecuteActivity(ctx, CreateNodeActivityName, activityInput)
		}

		errs := make([]error, len(futures))

		for i, future := range futures {
			errs[i] = errors.WrapIff(future.Get(ctx, nil), "creating node %q", input.Nodes[i].Name)
		}

		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	setClusterStatus(ctx, input.ClusterID, pkgCluster.Creating, "waiting for Kubernetes master") // nolint: errcheck

	if err := waitForMasterReadySignal(ctx, 1*time.Hour); err != nil {
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
			ConfigSecretID: configSecretID,
			Cluster: clustersetup.Cluster{
				ID:   input.ClusterID,
				UID:  input.ClusterUID,
				Name: input.ClusterName,
			},
			Organization: clustersetup.Organization{
				ID:   input.OrganizationID,
				Name: input.OrganizationName,
			},
		}

		future := workflow.ExecuteChildWorkflow(ctx, clustersetup.WorkflowName, workflowInput)
		if err := future.Get(ctx, nil); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	/*

		postHookWorkflowInput := cluster.RunPostHooksWorkflowInput{
			ClusterID: input.ClusterID,
			PostHooks: cluster.BuildWorkflowPostHookFunctions(input.PostHooks, true),
		}

		err = workflow.ExecuteChildWorkflow(ctx, cluster.RunPostHooksWorkflowName, postHookWorkflowInput).Get(ctx, nil)
		if err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}

	*/
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
