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

package workflow

import (
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	intPKEWorkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

const UpdateClusterWorkflowName = "pke-vsphere-update-cluster"

// UpdateClusterWorkflowInput
type UpdateClusterWorkflowInput struct {
	ClusterID         uint
	ClusterName       string
	ClusterUID        string
	OrganizationID    uint
	OrganizationName  string
	ResourcePoolName  string
	FolderName        string
	DatastoreName     string
	SecretID          string
	StorageSecretID   string
	K8sSecretID       string
	OIDCEnabled       bool
	MasterNodeNames   []string
	NodesToCreate     []Node
	NodesToDelete     []Node
	NodePoolsToDelete []NodePool
	HTTPProxy         intPKE.HTTPProxy
	NodePoolLabels    map[string]map[string]string
}

func UpdateClusterWorkflow(ctx workflow.Context, input UpdateClusterWorkflowInput) error {
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

	var httpProxy intPKEWorkflow.HTTPProxy
	{
		activityInput := intPKEWorkflow.AssembleHTTPProxySettingsActivityInput{
			OrganizationID:     input.OrganizationID,
			HTTPProxyHostPort:  getHostPort(input.HTTPProxy.HTTP),
			HTTPProxySecretID:  input.HTTPProxy.HTTP.SecretID,
			HTTPSProxyHostPort: getHostPort(input.HTTPProxy.HTTPS),
			HTTPSProxySecretID: input.HTTPProxy.HTTPS.SecretID,
		}
		var output intPKEWorkflow.AssembleHTTPProxySettingsActivityOutput
		if err := workflow.ExecuteActivity(ctx, intPKEWorkflow.AssembleHTTPProxySettingsActivityName, activityInput).Get(ctx, &output); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
		httpProxy = output.Settings
	}

	var masterIP string
	err := workflow.ExecuteActivity(ctx, GetPublicAddressActivityName, GetPublicAddressActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		NodeName:       input.MasterNodeNames[0],
	}).Get(ctx, &masterIP)
	if err != nil {
		return err
	}

	// set up node pool labels set
	{
		activityInput := clustersetup.ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: brn.New(input.OrganizationID, brn.SecretResourceType, input.K8sSecretID).String(),
			Labels:         input.NodePoolLabels,
		}
		err := workflow.ExecuteActivity(ctx, clustersetup.ConfigureNodePoolLabelsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", clustersetup.ConfigureNodePoolLabelsActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	// Create nodes
	{
		futures := make(map[string]workflow.Future)

		for _, node := range input.NodesToCreate {
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

	// Delete k8s nodes
	{
		futures := make(map[string]workflow.Future)

		for _, node := range input.NodesToDelete {
			activityInput := DeleteK8sNodeActivityInput{
				OrganizationID: input.OrganizationID,
				ClusterName:    input.ClusterName,
				K8sSecretID:    input.K8sSecretID,
				Name:           node.Name,
			}

			futures[node.Name] = workflow.ExecuteActivity(ctx, DeleteK8sNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, nil), "deleting kubernetes node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	// Delete VM's
	{
		futures := make(map[string]workflow.Future)

		for _, node := range input.NodesToDelete {
			activityInput := DeleteNodeActivityInput{
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				ClusterID:      input.ClusterID,
				ClusterName:    input.ClusterName,
				Node:           node,
			}

			futures[node.Name] = workflow.ExecuteActivity(ctx, DeleteNodeActivityName, activityInput)
		}

		errs := []error{}

		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, nil), "deleting node %q", i))
		}

		if err := errors.Combine(errs...); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	// Delete node pools
	if len(input.NodePoolsToDelete) > 0 {
		futures := make(map[string]workflow.Future)
		for _, np := range input.NodePoolsToDelete {
			workflowInput := DeleteNodePoolWorkflowInput{
				ClusterID:      input.ClusterID,
				ClusterName:    input.ClusterName,
				OrganizationID: input.OrganizationID,
				SecretID:       input.SecretID,
				K8sSecretID:    input.K8sSecretID,
				NodePool:       np,
			}

			futures[np.Name] = workflow.ExecuteChildWorkflow(ctx, DeleteNodePoolWorkflowName, workflowInput)
		}

		errs := []error{}
		for i := range futures {
			errs = append(errs, errors.WrapIff(futures[i].Get(ctx, nil), "deleting node pool %q", i))
		}
		if err := errors.Combine(errs...); err != nil {
			_ = setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	err = setClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage)
	if err != nil {
		_ = setClusterErrorStatus(ctx, input.ClusterID, err)
		return err
	}

	return nil
}
