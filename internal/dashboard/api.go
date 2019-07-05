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

package dashboard

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/cache"
)

// DashboardAPI implements the Dashboard API actions.
type DashboardAPI struct {
	clusterManager      *cluster.Manager
	clusterGroupManager *clustergroup.Manager
	logger              logrus.FieldLogger
	errorHandler        emperror.Handler
}

func NewDashboardAPI(
	clusterManager *cluster.Manager,
	clusterGroupManager *clustergroup.Manager,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) *DashboardAPI {
	return &DashboardAPI{
		clusterManager:      clusterManager,
		clusterGroupManager: clusterGroupManager,
		logger:              logger,
		errorHandler:        errorHandler,
	}
}

// @Summary Get Dashboard info for all clusters of an organization
// @Description returns dashboard metrics for selected/all clusters of an organization
// @Tags dashboard
// @Produce json
// @Param orgid path int true "Organization ID"
// @Success 200 {object} dashboard.GetDashboardResponse
// @Success 206 {object} dashboard.GetDashboardResponse
// @Failure 400 {object} common.ErrorResponse
// @Router /dashboard/{orgid}/clusters [get]
// @Security bearerAuth
func (d *DashboardAPI) GetDashboard(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	clusters, err := d.clusterManager.GetClusters(context.Background(), organizationID)
	if err != nil {
		d.logger.Errorf("error fetching clusters: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error listing clusters",
			Error:   err.Error(),
		})
		return
	}

	clusterResponseChan := make(chan ClusterInfo, len(clusters))
	defer close(clusterResponseChan)
	partialResponse := false

	i := 0
	for _, c := range clusters {
		status, err := c.GetStatus()
		if err == nil {
			if strings.ToUpper(status.Status) == "RUNNING" {
				go func() {
					logger := d.logger.WithField("clusterId", c.GetID())
					cluster, partial := d.getClusterDashboardInfo(logger, c, organizationID)
					if partial {
						partialResponse = true
					}
					clusterResponseChan <- cluster
				}()
				i++
			}
		}

	}

	clusterResponse := make([]ClusterInfo, 0)
	for j := 0; j < i; j++ {
		c := <-clusterResponseChan
		clusterResponse = append(clusterResponse, c)
	}

	if partialResponse {
		c.JSON(http.StatusPartialContent, GetDashboardResponse{Clusters: clusterResponse})
		return
	}
	c.JSON(http.StatusOK, GetDashboardResponse{Clusters: clusterResponse})

}

// @Summary Get Dashboard info for a cluster
// @Description returns dashboard metrics for selected cluster
// @Tags dashboard
// @Produce json
// @Param orgid path int true "Organization ID"
// @Param id path int true "C~luster ID"
// @Success 200 {object} dashboard.GetDashboardResponse
// @Success 206 {object} dashboard.GetDashboardResponse
// @Failure 400 {object} common.ErrorResponse
// @Router /dashboard/{orgid}/clusters/{id} [get]
// @Security bearerAuth
func (d *DashboardAPI) GetClusterDashboard(c *gin.Context) {
	clusterID, ok := ginutils.UintParam(c, "id")
	if !ok {
		return
	}

	organizationID := auth.GetCurrentOrganization(c.Request).ID

	cluster, err := d.clusterManager.GetClusterByID(context.Background(), organizationID, clusterID)
	if err != nil {
		d.logger.Errorf("cluster not found: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error fetching cluster",
			Error:   err.Error(),
		})
		return
	}

	logger := d.logger.WithField("clusterId", cluster.GetID())
	clusterInfo, partialResponse := d.getClusterDashboardInfo(logger, cluster, organizationID)
	if partialResponse {
		c.JSON(http.StatusPartialContent, clusterInfo)
		return
	}
	c.JSON(http.StatusOK, clusterInfo)
}

func createNodeInfoMap(pods []v1.Pod, nodes []v1.Node) map[string]*cache.NodeInfo {
	nodeInfoMap := make(map[string]*cache.NodeInfo)
	for _, pod := range pods {
		nodeName := pod.Spec.NodeName
		if len(nodeName) > 0 {
			if _, ok := nodeInfoMap[nodeName]; !ok {
				nodeInfoMap[nodeName] = cache.NewNodeInfo()
			}
			nodeInfoMap[nodeName].AddPod(pod.DeepCopy())
		}
	}
	return nodeInfoMap
}

func (d *DashboardAPI) getClusterDashboardInfo(logger *logrus.Entry, commonCluster cluster.CommonCluster, orgID uint) (clusterInfo ClusterInfo, partialResponse bool) {
	nodeStates := make([]Node, 0)
	clusterInfo = ClusterInfo{
		Name:         commonCluster.GetName(),
		Id:           fmt.Sprint(commonCluster.GetID()),
		Distribution: commonCluster.GetDistribution(),
		Cloud:        commonCluster.GetCloud(),
		Nodes:        nodeStates,
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		clusterInfo.Status = "ERROR"
		clusterInfo.StatusMessage = err.Error()
		partialResponse = true
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		clusterInfo.Status = "ERROR"
		clusterInfo.StatusMessage = err.Error()
		partialResponse = true
		return
	}

	clusterStatus, err := commonCluster.GetStatus()
	if err != nil {
		clusterInfo.Status = "ERROR"
		clusterInfo.StatusMessage = err.Error()
		partialResponse = true
		return
	}

	clusterInfo.Status = clusterStatus.Status
	clusterInfo.StatusMessage = clusterStatus.StatusMessage
	clusterInfo.CreatedAt = clusterStatus.CreatedAt
	clusterInfo.CreatorName = clusterStatus.CreatorName
	clusterInfo.CreatorId = clusterStatus.CreatorId
	clusterInfo.Region = clusterStatus.Region
	clusterInfo.Location = clusterStatus.Location
	clusterInfo.MasterVersion = clusterStatus.Version

	endPoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		d.logger.Warn(err.Error())
	} else {
		clusterInfo.Endpoint = endPoint
	}

	secret, err := commonCluster.GetSecretWithValidation()
	if err != nil {
		d.logger.Warn(err.Error())
	} else {
		clusterInfo.SecretName = secret.Name
	}

	clusterGroupName, err := d.clusterGroupManager.GetClusterGroupNameForCluster(commonCluster.GetID(), orgID)
	if err != nil {
		d.logger.Warn(err.Error())
	} else if clusterGroupName != nil {
		clusterInfo.ClusterGroup = *clusterGroupName
	}

	if aks, ok := commonCluster.(interface{ GetResourceGroupName() string }); ok {
		clusterInfo.ResourceGroup = aks.GetResourceGroupName()
	}

	if gke, ok := commonCluster.(interface{ GetProjectId() (string, error) }); ok {
		projectId, err := gke.GetProjectId()
		if err != nil {
			d.logger.WithField("clusterID", commonCluster.GetID()).Warnf("error while fetching project id for cluster: %s", err)
		} else {
			clusterInfo.Project = projectId
		}
	}

	clusterInfo.NodePools = make(map[string]NodePool, len(clusterStatus.NodePools))
	for name, nodePool := range clusterStatus.NodePools {
		clusterInfo.NodePools[name] = NodePool{
			Autoscaling:  nodePool.Autoscaling,
			Count:        nodePool.Count,
			InstanceType: nodePool.InstanceType,
			SpotPrice:    nodePool.SpotPrice,
			Preemptible:  nodePool.Preemptible,
			MinCount:     nodePool.MinCount,
			MaxCount:     nodePool.MaxCount,
			Image:        nodePool.Image,
			Version:      nodePool.Version,
			Labels:       nodePool.Labels,
			CreatedAt:    nodePool.CreatedAt,
			CreatorName:  nodePool.CreatorName,
			CreatorID:    nodePool.CreatorId,
		}
	}

	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		clusterInfo.Status = "ERROR"
		clusterInfo.StatusMessage = err.Error()
		partialResponse = true
		return
	}

	logger.Debug("List pods")
	podList, err := client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		clusterInfo.Status = "ERROR"
		clusterInfo.StatusMessage = err.Error()
		partialResponse = true
		return
	}

	nodeInfoMap := createNodeInfoMap(podList.Items, nodes.Items)

	clusterResourceRequestMap := make(map[v1.ResourceName]resource.Quantity)
	clusterResourceAllocatableMap := make(map[v1.ResourceName]resource.Quantity)

	for _, node := range nodes.Items {
		status := &Status{
			Allocatable: &Allocatable{},
			Capacity:    &Capacity{},
		}

		status.KernelDeadlock = fmt.Sprint(v1.ConditionUnknown)
		status.FrequentUnregisterNetDevice = fmt.Sprint(v1.ConditionUnknown)
		status.OutOfDisk = fmt.Sprint(v1.ConditionUnknown)
		status.MemoryPressure = fmt.Sprint(v1.ConditionUnknown)
		status.DiskPressure = fmt.Sprint(v1.ConditionUnknown)
		status.PIDPressure = fmt.Sprint(v1.ConditionUnknown)
		status.NetworkUnavailable = fmt.Sprint(v1.ConditionUnknown)

		for _, condition := range node.Status.Conditions {
			switch condition.Type {
			case v1.NodeReady:
				status.Ready = fmt.Sprint(condition.Status)
				status.LastHeartbeatTime = condition.LastHeartbeatTime.String()
			case v1.NodeOutOfDisk:
				status.OutOfDisk = fmt.Sprint(condition.Status)
			case v1.NodeMemoryPressure:
				status.MemoryPressure = fmt.Sprint(condition.Status)
			case v1.NodeDiskPressure:
				status.DiskPressure = fmt.Sprint(condition.Status)
			case v1.NodePIDPressure:
				status.PIDPressure = fmt.Sprint(condition.Status)
			case v1.NodeNetworkUnavailable:
				status.NetworkUnavailable = fmt.Sprint(condition.Status)
			}
		}

		status.CpuUsagePercent, status.Capacity.Cpu, status.Allocatable.Cpu = calculateNodeResourceUsage(v1.ResourceCPU, node, nodeInfoMap, clusterResourceRequestMap, clusterResourceAllocatableMap)
		status.MemoryUsagePercent, status.Capacity.Memory, status.Allocatable.Memory = calculateNodeResourceUsage(v1.ResourceMemory, node, nodeInfoMap, clusterResourceRequestMap, clusterResourceAllocatableMap)
		status.StorageUsagePercent, status.Capacity.EphemeralStorage, status.Allocatable.EphemeralStorage = calculateNodeResourceUsage(v1.ResourceEphemeralStorage, node, nodeInfoMap, clusterResourceRequestMap, clusterResourceAllocatableMap)
		status.Capacity.Pods = node.Status.Capacity.Pods().Value()
		status.Allocatable.Pods = node.Status.Allocatable.Pods().Value()

		instanceType, found := node.Labels["beta.kubernetes.io/instance-type"]
		if found {
			status.InstanceType = instanceType
		} else {
			status.InstanceType = "n/a"
		}

		nodeStates = append(nodeStates, Node{
			Name:              node.Name,
			CreationTimestamp: node.CreationTimestamp.String(),
			Status:            status,
		})
	}
	clusterInfo.Nodes = nodeStates
	clusterInfo.CpuUsagePercent = calculateClusterResourceUsage(v1.ResourceCPU, clusterResourceRequestMap, clusterResourceAllocatableMap)
	clusterInfo.MemoryUsagePercent = calculateClusterResourceUsage(v1.ResourceMemory, clusterResourceRequestMap, clusterResourceAllocatableMap)
	clusterInfo.StorageUsagePercent = calculateClusterResourceUsage(v1.ResourceEphemeralStorage, clusterResourceRequestMap, clusterResourceAllocatableMap)

	return
}

func calculateNodeResourceUsage(
	resourceName v1.ResourceName,
	node v1.Node,
	nodeInfoMap map[string]*cache.NodeInfo,
	clusterResourceRequestMap map[v1.ResourceName]resource.Quantity,
	clusterResourceAllocatableMap map[v1.ResourceName]resource.Quantity,
) (float64, string, string) {
	capacity, found := node.Status.Capacity[resourceName]
	if !found {
		return 0, "n/a", "n/a"
	}

	allocatable, found := node.Status.Allocatable[resourceName]
	if !found {
		return 0, "n/a", "n/a"
	}

	clusterResourceAllocatable, found := clusterResourceAllocatableMap[resourceName]
	if found {
		clusterResourceAllocatable.Add(allocatable)
		clusterResourceAllocatableMap[resourceName] = clusterResourceAllocatable
	} else {
		clusterResourceAllocatableMap[resourceName] = allocatable.DeepCopy()
	}

	podsRequest := resource.MustParse("0")
	nodeInfo := nodeInfoMap[node.Name]
	if nodeInfo != nil {
		for _, pod := range nodeInfo.Pods() {
			for _, container := range pod.Spec.Containers {
				if resourceValue, found := container.Resources.Requests[resourceName]; found {
					podsRequest.Add(resourceValue)
				}
			}
		}
	}

	clusterResourceRequest, found := clusterResourceRequestMap[resourceName]
	if found {
		clusterResourceRequest.Add(podsRequest)
		clusterResourceRequestMap[resourceName] = clusterResourceRequest
	} else {
		clusterResourceRequestMap[resourceName] = podsRequest.DeepCopy()
	}

	usagePercent := float64(podsRequest.MilliValue()) / float64(allocatable.MilliValue()) * 100

	if math.IsNaN(usagePercent) || math.IsInf(usagePercent, 0) {
		usagePercent = 0
	}

	return usagePercent, k8sutil.FormatResourceQuantity(resourceName, &capacity), k8sutil.FormatResourceQuantity(resourceName, &allocatable)
}

func calculateClusterResourceUsage(
	resourceName v1.ResourceName,
	clusterResourceRequestMap map[v1.ResourceName]resource.Quantity,
	clusterResourceAllocatableMap map[v1.ResourceName]resource.Quantity,
) float64 {
	clusterResourceRequest, found := clusterResourceRequestMap[resourceName]
	if !found {
		return 0
	}

	clusterResourceAllocatable, found := clusterResourceAllocatableMap[resourceName]
	if !found {
		return 0
	}

	usagePercent := float64(clusterResourceRequest.MilliValue()) / float64(clusterResourceAllocatable.MilliValue()) * 100
	if math.IsNaN(usagePercent) || math.IsInf(usagePercent, 0) {
		usagePercent = 0
	}

	return usagePercent
}
