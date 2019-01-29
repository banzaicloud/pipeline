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
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/cache"
)

type Allocatable struct {
	Cpu              string `json:"cpu"`
	EphemeralStorage string `json:"ephemeralStorage"`
	Memory           string `json:"memory"`
	Pods             int64  `json:"pods"`
}

type Capacity struct {
	Cpu              string `json:"cpu"`
	EphemeralStorage string `json:"ephemeralStorage"`
	Memory           string `json:"memory"`
	Pods             int64  `json:"pods"`
}

type Node struct {
	Name              string  `json:"name"`
	CreationTimestamp string  `json:"creationTimestamp"`
	Status            *Status `json:"status"`
}

type Status struct {
	Capacity                    *Capacity    `json:"capacity"`
	Allocatable                 *Allocatable `json:"allocatable"`
	Ready                       string       `json:"ready"`
	LastHeartbeatTime           string       `json:"lastHeartbeatTime"`
	FrequentUnregisterNetDevice string       `json:"frequentUnregisterNetDevice"`
	KernelDeadlock              string       `json:"kernelDeadlock"`
	NetworkUnavailable          string       `json:"networkUnavailable"`
	OutOfDisk                   string       `json:"outOfDisk"`
	MemoryPressure              string       `json:"memoryPressure"`
	DiskPressure                string       `json:"diskPressure"`
	PIDPressure                 string       `json:"pidPressure"`
	CpuUsagePercent             float64      `json:"cpuUsagePercent"`
	StorageUsagePercent         float64      `json:"storageUsagePercent"`
	MemoryUsagePercent          float64      `json:"memoryUsagePercent"`
	InstanceType                string       `json:"instanceType"`
}

type Cluster struct {
	Name                string    `json:"name"`
	Id                  string    `json:"id"`
	Status              string    `json:"status"`
	Distribution        string    `json:"distribution"`
	StatusMessage       string    `json:"statusMessage"`
	Cloud               string    `json:"cloud"`
	CreatedAt           time.Time `json:"createdAt"`
	Region              string    `json:"region"`
	Nodes               []Node    `json:"nodes"`
	CpuUsagePercent     float64   `json:"cpuUsagePercent"`
	StorageUsagePercent float64   `json:"storageUsagePercent"`
	MemoryUsagePercent  float64   `json:"memoryUsagePercent"`
}

// GetDashboardResponse Api object to be mapped to Get dashboard request
// swagger:model GetDashboardResponse
type GetDashboardResponse struct {
	Clusters []Cluster `json:"clusters"`
}

// GetProviderPathParams is a placeholder for the GetDashboard route path parameters
// swagger:parameters GetDashboard
type GetDashboardPathParams struct {
	// in:path
	OrgId string `json:"orgid"`
}

// swagger:route GET /dashboard/{orgid}/clusters orgid GetDashboard
//
// Returns returns dashboard metrics for selected/all clusters of an organization
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: GetDashboardResponse
// GetDashboard
func GetDashboard(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	logger := log.WithFields(logrus.Fields{
		"organization": organizationID,
	})

	// TODO: move these to a struct and create them only once upon application init
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(intCluster.NewClusters(config.DB()), secretValidator, cluster.NewNopClusterEvents(), nil, nil, nil, "", log, errorHandler)

	logger.Info("fetching clusters")

	clusters, err := clusterManager.GetClusters(context.Background(), organizationID)
	if err != nil {
		logger.Errorf("error listing clusters: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error listing clusters",
			Error:   err.Error(),
		})
		return
	}

	clusterResponseChan := make(chan Cluster, len(clusters))
	defer close(clusterResponseChan)

	i := 0
	for _, c := range clusters {
		status, err := c.GetStatus()
		if err == nil {
			if strings.ToUpper(status.Status) == "RUNNING" {
				logger := logger.WithField("cluster", c.GetName())
				go getClusterDashboard(logger, c, clusterResponseChan)
				i++
			}
		}

	}

	clusterResponse := make([]Cluster, 0)
	for j := 0; j < i; j++ {
		c := <-clusterResponseChan
		clusterResponse = append(clusterResponse, c)
	}

	c.JSON(http.StatusOK, GetDashboardResponse{Clusters: clusterResponse})

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

func getClusterDashboard(logger *logrus.Entry, commonCluster cluster.CommonCluster, clusterResponseChan chan Cluster) {
	nodeStates := make([]Node, 0)
	cluster := Cluster{
		Name:         commonCluster.GetName(),
		Id:           fmt.Sprint(commonCluster.GetID()),
		Distribution: commonCluster.GetDistribution(),
		Cloud:        commonCluster.GetCloud(),
		Nodes:        nodeStates,
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		cluster.Status = "ERROR"
		cluster.StatusMessage = err.Error()
		clusterResponseChan <- cluster
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		cluster.Status = "ERROR"
		cluster.StatusMessage = err.Error()
		clusterResponseChan <- cluster
		return
	}

	clusterStatus, err := commonCluster.GetStatus()
	if err != nil {
		cluster.Status = "ERROR"
		cluster.StatusMessage = err.Error()
		clusterResponseChan <- cluster
		return
	}

	cluster.Status = clusterStatus.Status
	cluster.CreatedAt = clusterStatus.CreatedAt
	cluster.Region = clusterStatus.Location

	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		cluster.Status = "ERROR"
		cluster.StatusMessage = err.Error()
		clusterResponseChan <- cluster
		return
	}

	log.Info("List pods")
	podList, err := client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		cluster.Status = "ERROR"
		cluster.StatusMessage = err.Error()
		clusterResponseChan <- cluster
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
	cluster.Nodes = nodeStates
	cluster.CpuUsagePercent = calculateClusterResourceUsage(v1.ResourceCPU, clusterResourceRequestMap, clusterResourceAllocatableMap)
	cluster.MemoryUsagePercent = calculateClusterResourceUsage(v1.ResourceMemory, clusterResourceRequestMap, clusterResourceAllocatableMap)
	cluster.StorageUsagePercent = calculateClusterResourceUsage(v1.ResourceEphemeralStorage, clusterResourceRequestMap, clusterResourceAllocatableMap)

	clusterResponseChan <- cluster

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
