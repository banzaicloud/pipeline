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

package driver

import (
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/utils"
)

func MakeClusterDeleter(events ClusterDeleterEvents, kubeProxyCache KubeProxyCache, logger Logger, secrets SecretStore, statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric, store pke.ClusterStore, workflowClient client.Client) ClusterDeleter {
	return ClusterDeleter{
		events:                     events,
		kubeProxyCache:             kubeProxyCache,
		logger:                     logger,
		secrets:                    secrets,
		statusChangeDurationMetric: statusChangeDurationMetric,
		store:                      store,
		workflowClient:             workflowClient,
	}
}

type ClusterDeleter struct {
	events                     ClusterDeleterEvents
	kubeProxyCache             KubeProxyCache
	logger                     Logger
	secrets                    SecretStore
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric
	store                      pke.ClusterStore
	workflowClient             client.Client
}

type SecretStore interface {
	Get(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
}

type ClusterDeleterEvents interface {
	ClusterDeleted(organizationID uint, clusterName string)
}

type KubeProxyCache interface {
	Delete(clusterUID string)
}

func (cd ClusterDeleter) DeleteCluster(ctx context.Context, clusterID uint, options cluster.DeleteClusterOptions) error {
	cl, err := cd.store.GetByID(clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to load cluster from data store")
	}
	return cd.Delete(ctx, cl, options.Force)
}

func (cd ClusterDeleter) Delete(ctx context.Context, cluster pke.PKEOnVsphereCluster, forced bool) error {
	logger := cd.logger.WithFields(map[string]interface{}{"clusterName": cluster.Name, "clusterID": cluster.ID, "forced": forced})
	logger.Info("Deleting cluster")

	masterVmNames, vmNames := getVMNames(cluster)
	masterNodes := make([]workflow.Node, 0)
	for _, vmName := range masterVmNames {
		masterNodes = append(masterNodes, workflow.Node{
			Name: vmName,
		})
	}
	nodes := make([]workflow.Node, 0)
	for _, vmName := range vmNames {
		nodes = append(nodes, workflow.Node{
			Name: vmName,
		})
	}

	input := workflow.DeleteClusterWorkflowInput{
		OrganizationID: cluster.OrganizationID,
		SecretID:       cluster.SecretID,
		ClusterID:      cluster.ID,
		ClusterName:    cluster.Name,
		ClusterUID:     cluster.UID,
		K8sSecretID:    cluster.K8sSecretID,
		Forced:         forced,
		MasterNodes:    masterNodes,
		Nodes:          nodes,
	}

	retryPolicy := &cadence.RetryPolicy{
		InitialInterval:    time.Second * 3,
		BackoffCoefficient: 2,
		ExpirationInterval: time.Minute * 3,
		MaximumAttempts:    5,
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
		RetryPolicy:                  retryPolicy,
	}

	if err := cd.store.SetStatus(cluster.ID, pkgCluster.Deleting, pkgCluster.DeletingMessage); err != nil {
		return errors.WrapIf(err, "failed to set cluster status")
	}

	timer, err := cd.getClusterStatusChangeDurationTimer(cluster)
	if err = errors.WrapIf(err, "failed to start status change duration metric timer"); err != nil {
		if forced {
			cd.logger.Error(err.Error())
			timer = metrics.NoopDurationMetricTimer{}
		} else {
			return err
		}
	}

	wfrun, err := cd.workflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.DeleteClusterWorkflowName, input)
	if err = errors.WrapIfWithDetails(err, "failed to start cluster deletion workflow", "cluster", cluster.Name); err != nil {
		_ = cd.store.SetStatus(cluster.ID, pkgCluster.Error, err.Error())
		return err
	}

	go func() {
		defer timer.RecordDuration()

		ctx := context.Background()

		if err := wfrun.Get(ctx, nil); err != nil {
			cd.logger.Error("cluster deleting workflow failed: " + err.Error())
			return
		}
		cd.kubeProxyCache.Delete(cluster.UID)
		if cd.events != nil {
			cd.events.ClusterDeleted(cluster.OrganizationID, cluster.Name)
		}
	}()

	if err = cd.store.SetActiveWorkflowID(cluster.ID, wfrun.GetID()); err != nil {
		return errors.WrapIfWithDetails(err, "failed to set active workflow ID for cluster", "cluster", cluster.Name, "workflowID", wfrun.GetID())
	}

	return nil
}

func (cd ClusterDeleter) getClusterStatusChangeDurationTimer(cluster pke.PKEOnVsphereCluster) (metrics.DurationMetricTimer, error) {
	if cd.statusChangeDurationMetric == nil {
		return metrics.NoopDurationMetricTimer{}, nil
	}

	values := metrics.ClusterStatusChangeDurationMetricValues{
		ProviderName: pkgCluster.Vsphere,
		LocationName: "na",
		Status:       pkgCluster.Deleting,
	}
	if global.Config.Telemetry.Debug {
		org, err := auth.GetOrganizationById(cluster.OrganizationID)
		if err != nil {
			return nil, errors.WrapIf(err, "Error during getting organization.")
		}
		values.OrganizationName = org.Name
		values.ClusterName = cluster.Name
	}
	return cd.statusChangeDurationMetric.StartTimer(values), nil
}

func (cd ClusterDeleter) DeleteByID(ctx context.Context, clusterID uint, forced bool) error {
	cl, err := cd.store.GetByID(clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to load cluster from data store")
	}
	return cd.Delete(ctx, cl, forced)
}

func getVMNames(cluster pke.PKEOnVsphereCluster) ([]string, []string) {
	masterVmNames := []string{}
	vmNames := []string{}
	for _, np := range cluster.NodePools {
		names := []string{}
		for j := 1; j <= np.Size; j++ {
			names = append(names, pke.GetVMName(cluster.Name, np.Name, j))
		}

		if utils.Contains(np.Roles, "master") {
			masterVmNames = append(masterVmNames, names...)
		} else {
			vmNames = append(vmNames, names...)
		}
	}
	return masterVmNames, vmNames
}
