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
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	intcluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

func NewEKSClusterDeleter(
	events ClusterDeleterEvents,
	kubeProxyCache KubeProxyCache,
	logger logrus.FieldLogger,
	secrets SecretStore,
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric,
	workflowClient client.Client,
	clusterGetter CommonClusterGetter,
) EKSClusterDeleter {
	return EKSClusterDeleter{
		events:                     events,
		kubeProxyCache:             kubeProxyCache,
		logger:                     logger,
		secrets:                    secrets,
		statusChangeDurationMetric: statusChangeDurationMetric,
		workflowClient:             workflowClient,
		clusterGetter:              clusterGetter,
	}
}

type CommonClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

type EKSClusterDeleter struct {
	events                     ClusterDeleterEvents
	kubeProxyCache             KubeProxyCache
	logger                     logrus.FieldLogger
	secrets                    SecretStore
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric
	workflowClient             client.Client
	clusterGetter              CommonClusterGetter
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

// DeleteCluster deletes an EKS Cluster
func (cd EKSClusterDeleter) DeleteCluster(ctx context.Context, clusterID uint, options intcluster.DeleteClusterOptions) error {
	var eksCluster *cluster.EKSCluster
	{
		cc, err := cd.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return errors.WrapIf(err, "failed to get cluster")
		}

		var ok bool
		if eksCluster, ok = cc.(*cluster.EKSCluster); !ok {
			return errors.NewWithDetails("not an EKS cluster", "clusterId", clusterID)
		}
	}

	logger := cd.logger.WithFields(logrus.Fields{
		"clusterName":    eksCluster.GetName(),
		"clusterID":      eksCluster.GetID(),
		"organizationID": eksCluster.GetOrganizationId(),
		"forced":         options.Force,
	})
	logger.Info("start deleting EKS Cluster")

	modelCluster := eksCluster.GetModel()

	nodePoolNames := make([]string, 0)
	for _, nodePool := range modelCluster.NodePools {
		nodePoolNames = append(nodePoolNames, nodePool.Name)
	}

	input := cluster.EKSDeleteClusterWorkflowInput{
		OrganizationID:      eksCluster.GetOrganizationId(),
		Region:              eksCluster.GetLocation(),
		SecretID:            eksCluster.GetSecretId(),
		ClusterID:           eksCluster.GetID(),
		ClusterUID:          eksCluster.GetUID(),
		ClusterName:         eksCluster.GetName(),
		NodePoolNames:       nodePoolNames,
		K8sSecretID:         eksCluster.GetConfigSecretId(),
		DefaultUser:         modelCluster.DefaultUser,
		Forced:              options.Force,
		GeneratedSSHKeyUsed: eksCluster.IsSSHGenerated(),
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 1 * 24 * time.Hour,
	}
	if err := eksCluster.SetStatus(pkgCluster.Deleting, pkgCluster.DeletingMessage); err != nil {
		return errors.WrapIf(err, "failed to set Cluster status")
	}

	timer, err := getClusterStatusChangeMetricTimer(eksCluster.GetCloud(), eksCluster.GetLocation(), pkgCluster.Deleting, eksCluster.GetOrganizationId(), eksCluster.GetName(), cd.statusChangeDurationMetric)
	if err = errors.WrapIf(err, "failed to start status change duration metric timer"); err != nil {
		if options.Force {
			logger.Error(err)
			timer = metrics.NoopDurationMetricTimer{}
		} else {
			return err
		}
	}

	wfrun, err := cd.workflowClient.ExecuteWorkflow(ctx, workflowOptions, cluster.EKSDeleteClusterWorkflowName, input)
	if err != nil {
		return err
	}

	go func() {
		defer timer.RecordDuration()

		ctx := context.Background()

		if err := wfrun.Get(ctx, nil); err != nil {
			logger.Error(errors.WrapIf(err, "cluster delete workflow failed"))
			return
		}
		cd.kubeProxyCache.Delete(eksCluster.GetUID())
		if cd.events != nil {
			cd.events.ClusterDeleted(eksCluster.GetOrganizationId(), eksCluster.GetName())
		}
	}()

	err = eksCluster.SetCurrentWorkflowID(wfrun.GetID())
	if err != nil {
		return err
	}

	return nil
}
