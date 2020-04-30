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
	"net"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	intPKEWorkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/pkg/brn"
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
	StorageSecretID  string
	OIDCEnabled      bool
	PostHooks        pkgCluster.PostHooks
	Nodes            []Node
	HTTPProxy        intPKE.HTTPProxy
	NodePoolLabels   map[string]map[string]string
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

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
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	var httpProxy intPKEWorkflow.HTTPProxy
	{
		activityInput := intPKEWorkflow.AssembleHTTPProxySettingsActivityInput{
			OrganizationID:     input.OrganizationID,
			HTTPProxyHostPort:  getHostPort(input.HTTPProxy.HTTP),
			HTTPProxySecretID:  input.HTTPProxy.HTTP.SecretID,
			HTTPProxyScheme:    input.HTTPProxy.HTTP.Scheme,
			HTTPSProxyHostPort: getHostPort(input.HTTPProxy.HTTPS),
			HTTPSProxySecretID: input.HTTPProxy.HTTPS.SecretID,
			HTTPSProxyScheme:   input.HTTPProxy.HTTPS.Scheme,
		}
		var output intPKEWorkflow.AssembleHTTPProxySettingsActivityOutput
		if err := workflow.ExecuteActivity(ctx, intPKEWorkflow.AssembleHTTPProxySettingsActivityName, activityInput).Get(ctx, &output); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
		httpProxy = output.Settings
	}

	var masterRef types.ManagedObjectReference
	// Create master nodes
	{
		futures := make(map[string]workflow.Future)

		for _, node := range input.Nodes {
			if !node.Master {
				continue
			}
			if node.UserDataScriptParams == nil {
				node.UserDataScriptParams = make(map[string]string)
			}

			node.UserDataScriptParams["HttpProxy"] = httpProxy.HTTPProxyURL
			node.UserDataScriptParams["HttpsProxy"] = httpProxy.HTTPSProxyURL

			activityInput := CreateNodeActivityInput{
				OrganizationID:   input.OrganizationID,
				SecretID:         input.SecretID,
				StorageSecretID:  input.StorageSecretID,
				ClusterID:        input.ClusterID,
				ClusterName:      input.ClusterName,
				ResourcePoolName: input.ResourcePoolName,
				FolderName:       input.FolderName,
				DatastoreName:    input.DatastoreName,
				Node:             node,
			}
			futures[node.Name] = workflow.ExecuteActivity(ctx, CreateNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, &masterRef), "creating node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	var masterIP string
	err := workflow.ExecuteActivity(ctx, WaitForIPActivityName, WaitForIPActivityInput{
		Ref:            masterRef,
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		ClusterName:    input.ClusterName,
	}).Get(ctx, &masterIP)
	if err != nil {
		return err
	}

	_ = setClusterStatus(ctx, input.ClusterID, pkgCluster.Creating, "waiting for Kubernetes master") // nolint: errcheck

	// Create worker nodes
	{
		futures := make(map[string]workflow.Future)

		for _, node := range input.Nodes {
			if node.Master {
				continue
			}
			if node.UserDataScriptParams == nil {
				node.UserDataScriptParams = make(map[string]string)
			}
			if node.UserDataScriptParams["PublicAddress"] == "" {
				node.UserDataScriptParams["PublicAddress"] = masterIP
			}

			node.UserDataScriptParams["HttpProxy"] = httpProxy.HTTPProxyURL
			node.UserDataScriptParams["HttpsProxy"] = httpProxy.HTTPSProxyURL

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
			futures[node.Name] = workflow.ExecuteActivity(ctx, CreateNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, nil), "creating node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

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
			ConfigSecretID: brn.New(input.OrganizationID, brn.SecretResourceType, configSecretID).String(),
			Cluster: clustersetup.Cluster{
				ID:   input.ClusterID,
				UID:  input.ClusterUID,
				Name: input.ClusterName,
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

func getHostPort(o intPKE.HTTPProxyOptions) string {
	if o.Host == "" {
		return ""
	}
	if o.Port == 0 {
		return o.Host
	}
	return net.JoinHostPort(o.Host, strconv.FormatUint(uint64(o.Port), 10))
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
