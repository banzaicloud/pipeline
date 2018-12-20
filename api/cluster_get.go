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

package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/banzaicloud/pipeline/internal/cluster/resourcesummary"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCluster fetches a K8S cluster in the cloud
func (a *ClusterAPI) GetCluster(c *gin.Context) {
	commonCluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if ok != true {
		return
	}

	errorHandler = emperror.HandlerWith(
		errorHandler,
		"clusterId", commonCluster.GetID(),
		"clusterName", commonCluster.GetName(),
	)

	clusterStatus, err := commonCluster.GetStatus()
	if err != nil {
		errorHandler.Handle(err)

		ginutils.ReplyWithErrorResponse(c, &common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return
	}

	response := GetClusterResponse{
		ID:            clusterStatus.ResourceID,
		Status:        clusterStatus.Status,
		StatusMessage: clusterStatus.StatusMessage,
		Name:          clusterStatus.Name,

		Region:       clusterStatus.Region,
		Location:     clusterStatus.Location,
		Cloud:        clusterStatus.Cloud,
		Distribution: clusterStatus.Distribution,
		Spot:         clusterStatus.Spot,

		Logging:      clusterStatus.Logging,
		Monitoring:   clusterStatus.Monitoring,
		SecurityScan: clusterStatus.SecurityScan,

		// TODO: keep one of the following?
		// TODO: is this correct?
		Version:       clusterStatus.Version,
		MasterVersion: clusterStatus.Version,

		NodePools: make(map[string]GetClusterNodePool, len(clusterStatus.NodePools)),

		CreatedAt:   clusterStatus.CreatedAt,
		CreatorName: clusterStatus.CreatorName,
		CreatorID:   clusterStatus.CreatorId,
	}

	for name, nodePool := range clusterStatus.NodePools {
		response.NodePools[name] = GetClusterNodePool{
			Autoscaling:  nodePool.Autoscaling,
			Count:        nodePool.Count,
			InstanceType: nodePool.InstanceType,
			SpotPrice:    nodePool.SpotPrice,
			Preemptible:  nodePool.Preemptible,
			MinCount:     nodePool.MinCount,
			MaxCount:     nodePool.MaxCount,
			Image:        nodePool.Image,
			Version:      nodePool.Version,

			CreatedAt:   nodePool.CreatedAt,
			CreatorName: nodePool.CreatorName,
			CreatorID:   nodePool.CreatorId,
		}
	}

	ready, err := commonCluster.IsReady()
	if err != nil {
		err = errors.WithMessage(err, "failed to check if the cluster is ready")
		errorHandler.Handle(err)
	}
	if err != nil || !ready { // Cluster is not ready yet or we can't check if it's ready
		c.JSON(http.StatusPartialContent, response)
		return
	}

	var partialResponse bool

	secret, err := commonCluster.GetSecretWithValidation()
	if err != nil {
		errorHandler.Handle(err)

		partialResponse = true
	} else {
		response.SecretID = secret.ID
		response.SecretName = secret.Name
	}

	endpoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		errorHandler.Handle(err)

		partialResponse = true
	} else {
		response.Endpoint = endpoint
	}

	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		errorHandler.Handle(err)

		// We cannot continue collecting data from this point
		c.JSON(http.StatusPartialContent, response)
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		errorHandler.Handle(err)

		// We cannot continue collecting data from this point
		c.JSON(http.StatusPartialContent, response)
		return
	}

	totalSummary, err := resourcesummary.GetTotalSummary(client)
	if err != nil {
		errorHandler.Handle(err)

		partialResponse = true
	} else {
		cpuResource := Resource(totalSummary.CPU)
		memoryResource := Resource(totalSummary.Memory)
		response.TotalSummary = &ResourceSummary{
			CPU:    &cpuResource,
			Memory: &memoryResource,
		}
	}

	for name, nodePool := range response.NodePools {
		selector := fmt.Sprintf("%s=%s", common.LabelKey, name)

		nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			errorHandler.Handle(errors.Wrap(err, "failed to get nodes"))

			partialResponse = true

			continue
		}

		nodePool.ResourceSummary = make(map[string]NodeResourceSummary, len(nodes.Items))

		for _, node := range nodes.Items {
			nodeSummary, err := resourcesummary.GetNodeSummary(client, node)
			if err != nil {
				errorHandler.Handle(errors.WithMessage(err, "failed to get node resource summary"))

				partialResponse = true

				continue
			}

			cpuResource := Resource(nodeSummary.CPU)
			memoryResource := Resource(nodeSummary.Memory)
			nodePool.ResourceSummary[node.Name] = NodeResourceSummary{
				ResourceSummary: ResourceSummary{
					CPU:    &cpuResource,
					Memory: &memoryResource,
				},
				Status: nodeSummary.Status,
			}
		}

		response.NodePools[name] = nodePool
	}

	if partialResponse {
		c.JSON(http.StatusPartialContent, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetClusterResponse contains the details of a cluster.
type GetClusterResponse struct {
	ID            uint   `json:"id"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage,omitempty"`
	Name          string `json:"name"`

	// If region not available fall back to Location
	Region       string `json:"region,omitempty"`
	Location     string `json:"location"`
	Cloud        string `json:"cloud"`
	Distribution string `json:"distribution"`
	Spot         bool   `json:"spot,omitempty"`

	Logging      bool `json:"logging"`
	Monitoring   bool `json:"monitoring"`
	SecurityScan bool `json:"securityscan"`

	// TODO: keep one of the following?
	Version       string `json:"version,omitempty"`
	MasterVersion string `json:"masterVersion,omitempty"`

	SecretID   string `json:"secretId"`
	SecretName string `json:"secretName"`

	Endpoint     string                        `json:"endpoint,omitempty"`
	NodePools    map[string]GetClusterNodePool `json:"nodePools,omitempty"`
	TotalSummary *ResourceSummary              `json:"totalSummary,omitempty"`

	CreatedAt   time.Time `json:"createdAt,omitempty"`
	CreatorName string    `json:"creatorName,omitempty"`
	CreatorID   uint      `json:"creatorId,omitempty"`
}

// GetClusterNodePool describes a cluster's node pool.
type GetClusterNodePool struct {
	Autoscaling     bool                           `json:"autoscaling"`
	Count           int                            `json:"count,omitempty"`
	InstanceType    string                         `json:"instanceType,omitempty"`
	SpotPrice       string                         `json:"spotPrice,omitempty"`
	Preemptible     bool                           `json:"preemptible,omitempty"`
	MinCount        int                            `json:"minCount,omitempty"`
	MaxCount        int                            `json:"maxCount,omitempty"`
	Image           string                         `json:"image,omitempty"`
	Version         string                         `json:"version,omitempty"`
	ResourceSummary map[string]NodeResourceSummary `json:"resourceSummary,omitempty"`

	CreatedAt   time.Time `json:"createdAt,omitempty"`
	CreatorName string    `json:"creatorName,omitempty"`
	CreatorID   uint      `json:"creatorId,omitempty"`
}

// ResourceSummary describes a node's resource summary with CPU and Memory capacity/request/limit/allocatable
type ResourceSummary struct {
	CPU    *Resource `json:"cpu,omitempty"`
	Memory *Resource `json:"memory,omitempty"`
}

type NodeResourceSummary struct {
	ResourceSummary

	Status string `json:"status,omitempty"`
}

// Resource describes a resource summary with capacity/request/limit/allocatable
type Resource struct {
	Capacity    string `json:"capacity,omitempty"`
	Allocatable string `json:"allocatable,omitempty"`
	Limit       string `json:"limit,omitempty"`
	Request     string `json:"request,omitempty"`
}
