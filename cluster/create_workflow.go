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
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const CreateClusterWorkflowName = "create-cluster-legacy"

type CreateClusterWorkflowInput struct {
	ClusterID        uint
	ClusterUID       string
	ClusterName      string
	OrganizationID   uint
	OrganizationName string
	Distribution     string
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

	if input.Distribution == pkgCluster.OKE {
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

		activityInput := SetupPrivilegesActivityInput{
			SecretID:  brn.New(input.OrganizationID, brn.SecretResourceType, configSecretID).String(),
			ClusterID: input.ClusterID,
		}

		err := workflow.ExecuteActivity(ctx, SetupPrivilegesActivityName, activityInput).Get(ctx, nil)
		if err != nil {
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
		}

		cwo := workflow.ChildWorkflowOptions{
			ExecutionStartToCloseTimeout: 30 * time.Minute,
			TaskStartToCloseTimeout:      40 * time.Minute,
		}
		ctx := workflow.WithChildOptions(ctx, cwo)

		future := workflow.ExecuteChildWorkflow(ctx, clustersetup.WorkflowName, workflowInput)
		if err := future.Get(ctx, nil); err != nil {
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

	if secretID := cluster.GetConfigSecretId(); secretID != "" {
		logger.Info("config is already present in Vault")

		return secretID, nil
	}

	activityInfo := activity.GetInfo(ctx)

	// On the first attempt try to get an existing config
	if activityInfo.Attempt == 0 {
		logger.Info("trying to get config for the first time")

		config, err := cluster.GetK8sConfig()
		if err == nil && len(config) > 0 {
			logger.Info("saving existing config")

			if err := StoreKubernetesConfig(cluster, config); err != nil {
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

		if err := StoreKubernetesConfig(cluster, config); err != nil {
			return "", err
		}

		return cluster.GetConfigSecretId(), nil
	}

	return cluster.GetConfigSecretId(), nil
}

const SetupPrivilegesActivityName = "setup-privileges-legacy"

type SetupPrivilegesActivityInput struct {
	ClusterID uint
	SecretID  string
}

type SetupPrivilegesActivity struct {
	clientFactory cluster.ClientFactory
	manager       *Manager
}

func NewSetupPrivilegesActivity(clientFactory cluster.ClientFactory, manager *Manager) SetupPrivilegesActivity {
	return SetupPrivilegesActivity{
		clientFactory: clientFactory,
		manager:       manager,
	}
}

func (a SetupPrivilegesActivity) Execute(ctx context.Context, input SetupPrivilegesActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With("clusterId", input.ClusterID)

	cluster, err := a.manager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	okeCluster, ok := cluster.(*OKECluster)
	if !ok {
		logger.Warn("not an OKE cluster")

		return nil
	}

	client, err := a.clientFactory.FromSecret(ctx, input.SecretID)
	if err != nil {
		return err
	}

	userName, err := okeCluster.GetKubernetesUserName()
	if err != nil {
		return err
	}

	name := "cluster-creator-admin-right"

	logger = logger.With("name", name, "user", userName)

	log.Info("creating cluster role")

	_, err = client.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: userName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin",
		},
	})
	if err != nil {
		return err
	}

	return nil
}
