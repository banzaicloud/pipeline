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
	"strings"
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// ClientFactory returns a Kubernetes client.
type ClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (kubernetes.Interface, error)
}

const CreateClusterWorkflowName = "create-cluster-legacy"

type CreateClusterWorkflowInput struct {
	ClusterID        uint
	ClusterUID       string
	ClusterName      string
	OrganizationID   uint
	OrganizationName string
	Distribution     string
	NodePoolLabels   map[string]map[string]string
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
				InitialInterval:          15 * time.Second,
				BackoffCoefficient:       1.0,
				MaximumAttempts:          30,
				NonRetriableErrorReasons: []string{"cadenceInternal:Panic", pkgCadence.ClientErrorReason},
			},
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		activityInput := DownloadK8sConfigActivityInput{
			ClusterID: input.ClusterID,
		}

		err := workflow.ExecuteActivity(ctx, DownloadK8sConfigActivityName, activityInput).Get(ctx, &configSecretID)
		if err != nil {
			return pkgCadence.UnwrapError(err)
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

	if input.Distribution == pkgCluster.ACK || input.Distribution == pkgCluster.AKS {
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

		activityInput := LabelNodesWithNodepoolNameActivityInput{
			SecretID:  brn.New(input.OrganizationID, brn.SecretResourceType, configSecretID).String(),
			ClusterID: input.ClusterID,
		}

		err := workflow.ExecuteActivity(ctx, LabelNodesWithNodepoolNameActivityName, activityInput).Get(ctx, nil)
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
			NodePoolLabels: input.NodePoolLabels,
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
			if strings.Contains(err.Error(), "PermissionDenied") {
				return "", pkgCadence.NewClientError(err)
			}
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
	clientFactory ClientFactory
	manager       *Manager
}

func NewSetupPrivilegesActivity(clientFactory ClientFactory, manager *Manager) SetupPrivilegesActivity {
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

const LabelNodesWithNodepoolNameActivityName = "label-nodes-with-nodepool-name-legacy"

type LabelNodesWithNodepoolNameActivityInput struct {
	ClusterID uint
	SecretID  string
}

type LabelNodesWithNodepoolNameActivity struct {
	clientFactory ClientFactory
	manager       *Manager
}

func NewLabelNodesWithNodepoolNameActivity(clientFactory ClientFactory, manager *Manager) LabelNodesWithNodepoolNameActivity {
	return LabelNodesWithNodepoolNameActivity{
		clientFactory: clientFactory,
		manager:       manager,
	}
}

type nodeNameLister interface {
	ListNodeNames() (map[string][]string, error)
}

func (a LabelNodesWithNodepoolNameActivity) Execute(ctx context.Context, input LabelNodesWithNodepoolNameActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With("clusterId", input.ClusterID)

	cluster, err := a.manager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	nodeNameLister, ok := cluster.(nodeNameLister)
	if !ok {
		logger.Warn("cluster does not expose node lists")

		return nil
	}

	client, err := a.clientFactory.FromSecret(ctx, input.SecretID)
	if err != nil {
		return err
	}

	nodeNames, err := nodeNameLister.ListNodeNames()
	if err != nil {
		return err
	}

	for poolName, nodes := range nodeNames {
		logger := logger.With("nodepool", poolName)
		logger.Debug("labeling nodepool")

		for _, nodeName := range nodes {
			logger := logger.With("node", nodeName)
			logger.Debug("labeling node")
			labels := map[string]string{pkgCommon.LabelKey: poolName}

			tokens := make([]string, 0, len(labels))
			for k, v := range labels {
				tokens = append(tokens, "\""+k+"\":\""+v+"\"")
			}
			labelString := "{" + strings.Join(tokens, ",") + "}"
			patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

			_, err = client.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, []byte(patch))

			if err != nil {
				logger.Warnf("error during adding label to node: %s", err.Error())
			}
		}
	}

	return nil
}
