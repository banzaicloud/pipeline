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

package cluster

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
)

const CreateClusterWorkflowName = "create-cluster-legacy"

type CreateClusterWorkflowInput struct {
	ClusterID        uint
	ClusterUID       string
	ClusterName      string
	OrganizationID   uint
	OrganizationName string
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {
	// Download k8s config (where applicable)
	var configSecretID string
	{
		ao := workflow.ActivityOptions{
			ScheduleToStartTimeout: 10 * time.Minute,
			StartToCloseTimeout:    20 * time.Minute,
			WaitForCancellation:    true,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:    15 * time.Second,
				BackoffCoefficient: 1.0,
				MaximumAttempts:    30,
			},
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		activityInput := DownloadK8sConfigActivityInput{
			ClusterID: input.ClusterID,
		}

		err := workflow.ExecuteActivity(ctx, DownloadK8sConfigActivityName, activityInput).Get(ctx, &configSecretID)
		if err != nil {
			return err
		}
	}

	{
		workflowInput := clustersetup.WorkflowInput{
			ConfigSecretID: fmt.Sprintf("brn:%d:secret:%s", input.OrganizationID, configSecretID),
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

		cwo := workflow.ChildWorkflowOptions{
			ExecutionStartToCloseTimeout: 30 * time.Minute,
			TaskStartToCloseTimeout:      40 * time.Minute,
		}
		ctx := workflow.WithChildOptions(ctx, cwo)

		err := workflow.ExecuteChildWorkflow(ctx, clustersetup.WorkflowName, workflowInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

const DownloadK8sConfigActivityName = "download-k8s-config-legacy"

type DownloadK8sConfigActivityInput struct {
	ClusterID uint
}

type DownloadK8sConfigActivity struct {
	manager *Manager
}

func NewDownloadK8sConfigActivity(manager *Manager) DownloadK8sConfigActivity {
	return DownloadK8sConfigActivity{
		manager: manager,
	}
}

// K8sConfigDownloader can download a cluster config.
type K8sConfigDownloader interface {
	DownloadK8sConfig() ([]byte, error)
}

func (a DownloadK8sConfigActivity) Execute(ctx context.Context, input DownloadK8sConfigActivityInput) (string, error) {
	logger := activity.GetLogger(ctx).Sugar().With("clusterId", input.ClusterID)

	cluster, err := a.manager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}

	if cluster.GetConfigSecretId() != "" {
		logger.Info("config is already present in Vault")

		return cluster.GetConfigSecretId(), nil
	}

	activityInfo := activity.GetInfo(ctx)

	// On the first attempt try to get an existing config
	if activityInfo.Attempt == 0 {
		logger.Info("trying to get config for the first time")

		config, err := cluster.GetK8sConfig()
		if err == nil && len(config) > 0 {
			logger.Info("saving existing config")

			err = StoreKubernetesConfig(cluster, config)
			if err != nil {
				return "", err
			}

			return cluster.GetConfigSecretId(), nil
		}
	}

	if downloader, ok := cluster.(K8sConfigDownloader); ok {
		logger.Info("attempting to download config")

		config, err := downloader.DownloadK8sConfig()
		if err != nil {
			return "", err
		}

		logger.Info("saving config")

		err = StoreKubernetesConfig(cluster, config)
		if err != nil {
			return "", err
		}

		return cluster.GetConfigSecretId(), nil
	}

	return cluster.GetConfigSecretId(), nil
}
