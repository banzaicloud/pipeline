// Copyright © 2018 Banzai Cloud
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

	"emperror.dev/errors"
	"go.uber.org/cadence/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

// DynamicClientFactory returns a dynamic Kubernetes client.
type DynamicClientFactory interface {
	// FromSecret creates a Kubernetes client for a cluster from a secret.
	FromSecret(ctx context.Context, secretID string) (dynamic.Interface, error)
}

type commonUpdater struct {
	request                  *cluster.UpdateClusterRequest
	clientFactory            DynamicClientFactory
	cluster                  CommonCluster
	userID                   uint
	scaleOptionsChanged      bool
	clusterPropertiesChanged bool
	workflowClient           client.Client
	externalBaseURL          string
	externalBaseURLInsecure  bool
	helmService              HelmService
}

type commonUpdateValidationError struct {
	msg string

	invalidRequest     bool
	preconditionFailed bool
}

func (e *commonUpdateValidationError) Error() string {
	return e.msg
}

func (e *commonUpdateValidationError) IsInvalid() bool {
	return e.invalidRequest
}

func (e *commonUpdateValidationError) IsPreconditionFailed() bool {
	return e.preconditionFailed
}

// NewCommonClusterUpdater returns a new cluster creator instance.
func NewCommonClusterUpdater(
	request *cluster.UpdateClusterRequest,
	clientFactory DynamicClientFactory,
	cluster CommonCluster,
	userID uint,
	workflowClient client.Client,
	externalBaseURL string,
	externalBaseURLInsecure bool,
	helmService HelmService,
) *commonUpdater {
	return &commonUpdater{
		request:                 request,
		clientFactory:           clientFactory,
		cluster:                 cluster,
		userID:                  userID,
		workflowClient:          workflowClient,
		externalBaseURL:         externalBaseURL,
		externalBaseURLInsecure: externalBaseURLInsecure,
		helmService:             helmService,
	}
}

// Validate implements the clusterUpdater interface.
func (c *commonUpdater) Validate(ctx context.Context) error {
	if c.cluster.GetCloud() != c.request.Cloud {
		return &commonUpdateValidationError{
			msg:            fmt.Sprintf("cloud provider [%s] does not match the cluster's cloud provider [%s]", c.request.Cloud, c.cluster.GetCloud()),
			invalidRequest: true,
		}
	}

	status, err := c.cluster.GetStatus()
	if err != nil {
		return errors.WrapIf(err, "could not get cluster status")
	}

	if status.Status != cluster.Running && status.Status != cluster.Warning {
		return errors.WithDetails(
			&commonUpdateValidationError{
				msg:                fmt.Sprintf("cluster is not in %s or %s state yet", cluster.Running, cluster.Warning),
				preconditionFailed: true,
			},
			"status", status.Status,
		)
	}

	return nil
}

// Prepare implements the clusterUpdater interface.
func (c *commonUpdater) Prepare(ctx context.Context) (CommonCluster, error) {
	c.cluster.AddDefaultsToUpdate(c.request)

	c.scaleOptionsChanged = isDifferent(c.request.ScaleOptions, c.cluster.GetScaleOptions()) == nil
	c.clusterPropertiesChanged = true

	if err := c.cluster.CheckEqualityToUpdate(c.request); err != nil {
		c.clusterPropertiesChanged = false
		if !c.scaleOptionsChanged {
			return nil, &commonUpdateValidationError{
				msg:            err.Error(),
				invalidRequest: true,
			}
		}
	}

	if err := c.request.Validate(); err != nil {
		return nil, &commonUpdateValidationError{
			msg:            err.Error(),
			invalidRequest: true,
		}
	}

	if err := c.cluster.SetStatus(cluster.Updating, cluster.UpdatingMessage); err != nil {
		return nil, err
	}
	return c.cluster, c.cluster.Persist()
}

func buildNodePoolsLabelList(commonCluster CommonCluster, updateRequest *cluster.UpdateClusterRequest) ([]NodePoolLabels, error) {
	// we need to retrieve existing node pools, as update request doesn't necessary contains instanceType, spot price etc.
	clStatus, err := commonCluster.GetStatus()
	if err != nil {
		return nil, err
	}
	existingNodePoolMap := clStatus.NodePools

	nodePools := make([]NodePoolLabels, 0)
	cloudType := updateRequest.Cloud

	if updateRequest.PKE != nil {
		for name, np := range updateRequest.PKE.NodePools {
			npls := NodePoolLabels{
				NodePoolName: name,
				Existing:     false,
				InstanceType: np.InstanceType,
				SpotPrice:    np.SpotPrice,
				CustomLabels: np.Labels,
			}
			existingNodePool, ok := existingNodePoolMap[name]
			if ok {
				npls.Existing = true
				npls.InstanceType = existingNodePool.InstanceType
				npls.SpotPrice = existingNodePool.SpotPrice
			}
			nodePools = append(nodePools, npls)
		}
		return nodePools, nil
	}

	switch cloudType {
	case cluster.Azure:
		for name, np := range updateRequest.AKS.NodePools {
			if np != nil {
				npls := NodePoolLabels{
					NodePoolName: name,
					Existing:     false,
					CustomLabels: np.Labels,
				}
				existingNodePool, ok := existingNodePoolMap[name]
				if ok {
					npls.Existing = true
					npls.InstanceType = existingNodePool.InstanceType
				}
				nodePools = append(nodePools, npls)
			}
		}

	case cluster.Google:
		for name, np := range updateRequest.GKE.NodePools {
			if np != nil {
				npls := NodePoolLabels{
					NodePoolName: name,
					Existing:     false,
					InstanceType: np.NodeInstanceType,
					Preemptible:  np.Preemptible,
					CustomLabels: np.Labels,
				}
				existingNodePool, ok := existingNodePoolMap[name]
				if ok {
					npls.Existing = true
					npls.InstanceType = existingNodePool.InstanceType
					npls.Preemptible = existingNodePool.Preemptible
				}
				nodePools = append(nodePools, npls)
			}
		}

	case cluster.Kubernetes:
	}

	return nodePools, nil
}

// Update implements the clusterUpdater interface.
func (c *commonUpdater) Update(ctx context.Context) error {
	if c.scaleOptionsChanged {
		c.cluster.SetScaleOptions(c.request.ScaleOptions)
	}

	if !c.clusterPropertiesChanged && !c.scaleOptionsChanged {
		return nil
	}

	// obtain desired set of labels for each node pool
	nodePoolLabelsList, err := buildNodePoolsLabelList(c.cluster, c.request)
	if err != nil {
		return err
	}
	labelsMap, err := GetDesiredLabelsForCluster(ctx, c.cluster, nodePoolLabelsList)
	if err != nil {
		return err
	}

	secretID := brn.New(c.cluster.GetOrganizationId(), brn.SecretResourceType, c.cluster.GetConfigSecretId()).String()
	dclient, err := c.clientFactory.FromSecret(ctx, secretID)
	if err != nil {
		return err
	}

	manager := npls.NewManager(dclient, global.Config.Cluster.Labels.Namespace)

	if err = manager.Sync(ctx, labelsMap); err != nil {
		return err
	}

	if updater, ok := c.cluster.(interface {
		UpdatePKECluster(context.Context, *cluster.UpdateClusterRequest, uint, client.Client, string, bool) error
	}); ok {
		err = updater.UpdatePKECluster(ctx, c.request, c.userID, c.workflowClient, c.externalBaseURL, c.externalBaseURLInsecure)
	} else {
		err = c.cluster.UpdateCluster(c.request, c.userID)
	}
	if err != nil {
		return err
	}

	if err := DeployClusterAutoscaler(c.cluster, c.helmService); err != nil {
		return errors.WrapIf(err, "deploying cluster autoscaler failed")
	}

	// on certain clouds like Alibaba we still need to add node pool name labels
	if err := labelNodesWithNodePoolName(ctx, c.cluster); err != nil {
		return errors.WrapIf(err, "adding labels to nodes failed")
	}
	return nil
}

// labelNodesWithNodePoolName add node pool name labels for all nodes.
// It's used only used in case of ACK etc. when we're not able to add labels via API.
func labelNodesWithNodePoolName(ctx context.Context, commonCluster CommonCluster) error {
	switch commonCluster.GetDistribution() {
	case cluster.EKS, cluster.GKE, cluster.PKE, cluster.AKS:
		log.Infof("nodes are already labelled on : %v", commonCluster.GetDistribution())
		return nil
	}

	nodeNameLister, ok := commonCluster.(nodeNameLister)
	if !ok {
		log.Debug("cluster does not expose node names")

		return nil
	}

	log.Debug("get K8S config")
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		return err
	}

	log.Debug("get K8S connection")
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}

	log.Debug("list node names")
	nodeNames, err := nodeNameLister.ListNodeNames()
	if err != nil {
		return err
	}

	for poolName, nodes := range nodeNames {
		log.Debugf("nodepool: [%s]", poolName)
		for _, nodeName := range nodes {
			log.Infof("add label to node [%s]", nodeName)
			labels := map[string]string{pkgCommon.LabelKey: poolName}

			if err := addLabelsToNode(ctx, client, nodeName, labels); err != nil {
				log.Warnf("error during adding label to node [%s]: %s", nodeName, err.Error())
			}
		}
	}

	log.Info("add labels finished")

	return nil
}

// addLabelsToNode add label to the given node
func addLabelsToNode(ctx context.Context, client *kubernetes.Clientset, nodeName string, labels map[string]string) (err error) {
	tokens := make([]string, 0, len(labels))
	for k, v := range labels {
		tokens = append(tokens, "\""+k+"\":\""+v+"\"")
	}
	labelString := "{" + strings.Join(tokens, ",") + "}"
	patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

	_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	return
}
