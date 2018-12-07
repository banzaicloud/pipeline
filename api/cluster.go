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
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/pkg/providers"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	resourceHelper "k8s.io/kubernetes/pkg/api/v1/resource"
)

const (
	statusReady    = "Ready"
	statusNotReady = "Not ready"
	statusUnknown  = "Unknown"
	readyTrue      = "True"
	readyFalse     = "False"
)

const (
	zeroCPU    = "0 CPU"
	zeroMemory = "0 B"
)

// ClusterAPI implements the Cluster API actions.
type ClusterAPI struct {
	clusterManager *cluster.Manager

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewClusterAPI returns a new ClusterAPI instance.
func NewClusterAPI(clusterManager *cluster.Manager, logger logrus.FieldLogger, errorHandler emperror.Handler) *ClusterAPI {
	return &ClusterAPI{
		clusterManager: clusterManager,

		logger:       logger,
		errorHandler: errorHandler,
	}
}

// getClusterFromRequest just a simple getter to build commonCluster object this handles error messages directly
// Deprecated: use internal.clusterGetter instead
func getClusterFromRequest(c *gin.Context) (cluster.CommonCluster, bool) {
	// TODO: move these to a struct and create them only once upon application init
	clusters := intCluster.NewClusters(config.DB())
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(clusters, secretValidator, cluster.NewNopClusterEvents(), nil, nil, log, errorHandler)
	clusterGetter := common.NewClusterGetter(clusterManager, log, errorHandler)

	return clusterGetter.GetClusterFromRequest(c)
}

func getPostHookFunctions(postHooks pkgCluster.PostHooks) (ph []cluster.PostFunctioner) {

	log.Info("Get posthook function(s)")
	var securityScanPosthook cluster.PostFunctioner

	for postHookName, param := range postHooks {

		function := cluster.HookMap[postHookName]
		if function != nil {

			if f, isOk := function.(*cluster.PostFunctionWithParam); isOk {
				fa := *f
				fa.SetParams(param)
				function = &fa
			}

			log.Infof("posthook function: %s", function)
			log.Infof("posthook params: %#v", param)
			if postHookName == pkgCluster.InstallAnchoreImageValidator {
				securityScanPosthook = function
			} else {
				ph = append(ph, function)
			}
		} else {
			log.Warnf("there's no function with this name [%s]", postHookName)
		}
	}
	if securityScanPosthook != nil {
		ph = append(ph, securityScanPosthook)
	}

	log.Infof("Found posthooks: %v", ph)

	return
}

// GetClusterStatus retrieves the cluster status
func GetClusterStatus(c *gin.Context) {

	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	response, err := commonCluster.GetStatus()
	if err != nil {
		log.Errorf("Error during getting status: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting status",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response)
	return
}

// GetClusterConfig gets a cluster config
func GetClusterConfig(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}
	config, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting config",
			Error:   err.Error(),
		})
		return
	}

	contentType := c.NegotiateFormat(gin.MIMEPlain, gin.MIMEJSON)
	log.Debug("Content-Type: ", contentType)
	switch contentType {
	case gin.MIMEJSON:
		c.JSON(http.StatusOK, pkgCluster.GetClusterConfigResponse{
			Status: http.StatusOK,
			Data:   string(config),
		})
	default:
		c.String(http.StatusOK, string(config))
	}
	return
}

// GetApiEndpoint returns the Kubernetes Api endpoint
func GetApiEndpoint(c *gin.Context) {

	log.Info("Start getting API endpoint")

	log.Info("Create common cluster model from request")
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	log.Info("Start getting API endpoint")
	endPoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		log.Errorf("Error during getting api endpoint: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting endpoint",
			Error:   err.Error(),
		})
		return
	}

	log.Debugf("API endpoint: %s", endPoint)

	c.String(http.StatusOK, endPoint)
	return
}

// GetClusters fetches all the K8S clusters from the cloud.
func (a *ClusterAPI) GetClusters(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	logger := a.logger.WithFields(logrus.Fields{
		"organization": organizationID,
	})

	logger.Info("fetching clusters")

	clusters, err := a.clusterManager.GetClusters(context.Background(), organizationID)
	if err != nil {
		logger.Errorf("error listing clusters: %s", err.Error())

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error listing clusters",
			Error:   err.Error(),
		})

		return
	}

	response := make([]pkgCluster.GetClusterStatusResponse, 0)

	for _, c := range clusters {
		logger := logger.WithField("cluster", c.GetName())

		status, err := c.GetStatus()
		if err != nil {
			//TODO we want skip or return error?
			logger.Errorf("get cluster status failed: %s", err.Error())
		} else {
			response = append(response, *status)
		}
	}

	c.JSON(http.StatusOK, response)
}

// ReRunPostHooks handles {cluster_id}/posthooks API request
func ReRunPostHooks(c *gin.Context) {

	log.Info("Get common cluster")
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	var ph pkgCluster.PostHooks
	if err := c.BindJSON(&ph); err != nil {
		log.Errorf("error during binding request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error during binding request",
			Error:   err.Error(),
		})
		return
	}

	var posthooks []cluster.PostFunctioner
	if len(ph) == 0 {
		posthooks = cluster.BasePostHookFunctions
	} else {
		posthooks = getPostHookFunctions(ph)
	}

	log.Infof("Cluster id: %d", commonCluster.GetID())
	log.Infof("Run posthook(s): %v", posthooks)

	go cluster.RunPostHooks(posthooks, commonCluster)

	c.Status(http.StatusOK)
}

// ClusterHEAD checks the cluster ready
func ClusterHEAD(c *gin.Context) {

	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	log.Info("getting cluster")
	_, err := commonCluster.GetClusterDetails()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	c.Status(http.StatusOK)

}

// GetPodDetails returns all pods with details
func GetPodDetails(c *gin.Context) {

	commonCluster, isOk := getClusterFromRequest(c)
	if !isOk {
		return
	}

	response, err := describePods(commonCluster)
	if err != nil {
		log.Errorf("Error during getting pod details: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting pod details",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)

}

func describePods(commonCluster cluster.CommonCluster) (items []pkgCluster.PodDetailsResponse, err error) {

	log.Info("get K8S config")
	var kubeConfig []byte
	kubeConfig, err = commonCluster.GetK8sConfig()
	if err != nil {
		return
	}

	log.Info("get k8S connection")
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return
	}

	log.Info("list pods")
	var pods []v1.Pod
	pods, err = listPods(client, "", "")
	if err != nil {
		return
	}

	log.Infof("pods: %d", len(pods))

	for _, pod := range pods {
		req, limits := calculatePodsTotalRequestsAndLimits([]v1.Pod{pod})

		summary := getResourceSummary(nil, nil, req, limits)

		items = append(items, pkgCluster.PodDetailsResponse{
			Name:          pod.Name,
			Namespace:     pod.Namespace,
			CreatedAt:     pod.CreationTimestamp.Time,
			Labels:        pod.Labels,
			RestartPolicy: string(pod.Spec.RestartPolicy),
			Conditions:    pod.Status.Conditions,
			Summary:       summary,
		})
	}

	return

}

// GetClusterDetails fetch a K8S cluster in the cloud
func GetClusterDetails(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}
	log.Info("getting cluster details")
	details, err := commonCluster.GetClusterDetails()
	if err != nil {
		log.Errorf("Error getting cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting cluster",
			Error:   err.Error(),
		})
		return
	}

	log.Info("Start getting API endpoint")
	endpoint, err := commonCluster.GetAPIEndpoint()
	if err != nil {
		log.Warnf("Error during getting API endpoint: %s", err.Error())
	}
	details.Endpoint = endpoint

	log.Info("Add resource summary to node(s)")
	if err := addResourceSummaryToDetails(commonCluster, details); err != nil {
		log.Warnf("Error during adding summary: %s", err.Error())
	}

	secret, err := commonCluster.GetSecretWithValidation()
	if err != nil {
		log.Errorf("Error getting cluster secret: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error getting cluster secret",
			Error:   err.Error(),
		})
		return
	}

	details.SecretId = secret.ID
	details.SecretName = secret.Name

	c.JSON(http.StatusOK, details)
}

// addResourceSummaryToDetails adds resource summary to all node in each pool
func addResourceSummaryToDetails(commonCluster cluster.CommonCluster, details *pkgCluster.DetailsResponse) error {

	log.Info("get K8S config")
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		return err
	}

	log.Info("get k8S connection")
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}

	// add node summary
	log.Info("Add summary to nodes")
	for name := range details.NodePools {

		if err := addNodeSummaryToDetails(client, details, name); err != nil {
			return err
		}

	}

	// add total summary
	log.Info("add total summary")
	return addTotalSummaryToDetails(client, details)
}

// addTotalSummaryToDetails calculate all resource summary
func addTotalSummaryToDetails(client *kubernetes.Clientset, details *pkgCluster.DetailsResponse) (err error) {

	log.Info("list nodes")
	var nodeList *v1.NodeList
	nodeList, err = client.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return
	}

	log.Infof("nodes [%d]", len(nodeList.Items))

	log.Info("list pods")
	var pods []v1.Pod
	pods, err = listPods(client, "", "")
	if err != nil {
		return
	}

	log.Infof("pods [%d]", len(pods))

	log.Info("Calculate total requests/limits/capacity/allocatable")
	requests, limits := calculatePodsTotalRequestsAndLimits(pods)
	capacity, allocatable := calculateNodesTotalCapacityAndAllocatable(nodeList.Items)

	resourceSummary := getResourceSummary(capacity, allocatable, requests, limits)
	details.TotalSummary = resourceSummary

	return
}

// addNodeSummaryToDetails adds node resource summary
func addNodeSummaryToDetails(client *kubernetes.Clientset, details *pkgCluster.DetailsResponse, nodePoolName string) error {

	selector := fmt.Sprintf("%s=%s", pkgCommon.LabelKey, nodePoolName)

	log.Infof("List nodes with selector: %s", selector)

	nodes, err := client.CoreV1().Nodes().List(meta_v1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}

	log.Infof("nodes [%d]", len(nodes.Items))

	details.NodePools[nodePoolName].ResourceSummary = make(map[string]pkgCluster.ResourceSummary)

	for _, node := range nodes.Items {

		log.Infof("add summary to node [%s] in nodepool [s]", node.Name, nodePoolName)

		resourceSummary, err := getResourceSummaryFromNode(client, &node)
		if err != nil {
			return err
		}
		details.NodePools[nodePoolName].ResourceSummary[node.Name] = *resourceSummary
		log.Infof("summary added to node [%s] in nodepool [%s]", node.Name, nodePoolName)
	}

	details.NodePools[nodePoolName].Count = len(nodes.Items)

	return nil
}

// getResourceSummaryFromNode return resource summary for the given node
func getResourceSummaryFromNode(client *kubernetes.Clientset, node *v1.Node) (*pkgCluster.ResourceSummary, error) {

	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name)
	if err != nil {
		return nil, err
	}

	log.Infof("start getting requests and limits of all pods in all namespace")
	requests, limits, err := getAllPodsRequestsAndLimitsInAllNamespace(client, fieldSelector.String())
	if err != nil {
		return nil, err
	}

	var capCPU resource.Quantity
	var capMemory resource.Quantity
	var allocCPU resource.Quantity
	var allocMemory resource.Quantity
	if cpu := node.Status.Capacity.Cpu(); cpu != nil {
		capCPU = *cpu
	}

	if mem := node.Status.Capacity.Memory(); mem != nil {
		capMemory = *mem
	}

	if cpu := node.Status.Allocatable.Cpu(); cpu != nil {
		allocCPU = *cpu
	}

	if mem := node.Status.Allocatable.Memory(); mem != nil {
		allocMemory = *mem
	}

	// set capacity map
	capacity := map[v1.ResourceName]resource.Quantity{
		v1.ResourceCPU:    capCPU,
		v1.ResourceMemory: capMemory,
	}

	// set allocatable map
	allocatable := map[v1.ResourceName]resource.Quantity{
		v1.ResourceCPU:    allocCPU,
		v1.ResourceMemory: allocMemory,
	}

	resourceSummary := getResourceSummary(capacity, allocatable, requests, limits)
	resourceSummary.Status = getNodeStatus(node)

	return resourceSummary, nil

}

// getNodeStatus returns the node actual status
func getNodeStatus(node *v1.Node) string {

	for _, condition := range node.Status.Conditions {
		if condition.Type == statusReady {
			switch condition.Status {
			case readyTrue:
				return statusReady
			case readyFalse:
				return statusNotReady
			default:
				return statusUnknown

			}
		}
	}

	return ""

}

// getResourceSummary returns ResourceSummary type with the given data
func getResourceSummary(capacity, allocatable, requests, limits map[v1.ResourceName]resource.Quantity) *pkgCluster.ResourceSummary {

	var capMem = zeroMemory
	var capCPU = zeroCPU
	var allMem = zeroMemory
	var allCPU = zeroCPU
	var reqMem = zeroMemory
	var reqCPU = zeroCPU
	var limitMem = zeroMemory
	var limitCPU = zeroCPU

	if cpu, ok := capacity[v1.ResourceCPU]; ok {
		capCPU = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &cpu)
	}

	if memory, ok := capacity[v1.ResourceMemory]; ok {
		capMem = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &memory)
	}

	if cpu, ok := allocatable[v1.ResourceCPU]; ok {
		allCPU = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &cpu)
	}

	if memory, ok := allocatable[v1.ResourceMemory]; ok {
		allMem = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &memory)
	}

	if value, ok := requests[v1.ResourceCPU]; ok {
		reqCPU = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &value)
	}

	if value, ok := requests[v1.ResourceMemory]; ok {
		reqMem = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &value)
	}

	if value, ok := limits[v1.ResourceCPU]; ok {
		limitCPU = k8sutil.FormatResourceQuantity(v1.ResourceCPU, &value)
	}

	if value, ok := limits[v1.ResourceMemory]; ok {
		limitMem = k8sutil.FormatResourceQuantity(v1.ResourceMemory, &value)
	}

	return &pkgCluster.ResourceSummary{
		Cpu: &pkgCluster.CPU{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem{
				Capacity:    capCPU,
				Allocatable: allCPU,
				Limit:       limitCPU,
				Request:     reqCPU,
			},
		},
		Memory: &pkgCluster.Memory{
			ResourceSummaryItem: pkgCluster.ResourceSummaryItem{
				Capacity:    capMem,
				Allocatable: allMem,
				Limit:       limitMem,
				Request:     reqMem,
			},
		},
	}
}

func getAllPodsRequestsAndLimitsInAllNamespace(client *kubernetes.Clientset, fieldSelector string) (map[v1.ResourceName]resource.Quantity, map[v1.ResourceName]resource.Quantity, error) {

	log.Infof("list pods with field selector: %s", fieldSelector)
	podList, err := listPods(client, fieldSelector, "")
	if err != nil {
		return nil, nil, err
	}

	log.Infof("pods [%d]", len(podList))
	log.Infof("calculate requests and limits")
	req, limits := calculatePodsTotalRequestsAndLimits(podList)
	return req, limits, nil
}

// listPods returns list of pods in all namespaces
func listPods(client *kubernetes.Clientset, fieldSelector string, labelSelector string) ([]v1.Pod, error) {

	log := log.WithFields(logrus.Fields{
		"fieldSelector": fieldSelector,
		"labelSelector": labelSelector,
	})

	log.Info("List pods")
	podList, err := client.CoreV1().Pods("").List(meta_v1.ListOptions{
		FieldSelector: fieldSelector,
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// calculateNodesTotalCapacityAndAllocatable calculates capacity and allocatable of the given nodes
func calculateNodesTotalCapacityAndAllocatable(nodeList []v1.Node) (caps map[v1.ResourceName]resource.Quantity, allocs map[v1.ResourceName]resource.Quantity) {

	caps, allocs = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, node := range nodeList {

		nodeCaps, nodeAllocs := nodeCapacityAndAllocatable(&node)
		for nodeCapName, nodeCapValue := range nodeCaps {
			if value, ok := caps[nodeCapName]; !ok {
				caps[nodeCapName] = *nodeCapValue.Copy()
			} else {
				value.Add(nodeCapValue)
				caps[nodeCapName] = value
			}
		}

		for nodeAllocName, nodeAllocValue := range nodeAllocs {
			if value, ok := allocs[nodeAllocName]; !ok {
				allocs[nodeAllocName] = *nodeAllocValue.Copy()
			} else {
				value.Add(nodeAllocValue)
				allocs[nodeAllocName] = value
			}
		}
	}

	return
}

// nodeCapacityAndAllocatable returns the given node's capacity and allocatable
func nodeCapacityAndAllocatable(node *v1.Node) (caps map[v1.ResourceName]resource.Quantity, allocs map[v1.ResourceName]resource.Quantity) {
	caps, allocs = make(map[v1.ResourceName]resource.Quantity), make(map[v1.ResourceName]resource.Quantity)

	nodeCap := node.Status.Capacity
	nodeAlloc := node.Status.Allocatable

	if nodeCap.Memory() != nil {
		caps[v1.ResourceMemory] = *nodeCap.Memory()
	}

	if nodeCap.Cpu() != nil {
		caps[v1.ResourceCPU] = *nodeCap.Cpu()
	}

	if nodeAlloc.Memory() != nil {
		allocs[v1.ResourceMemory] = *nodeAlloc.Memory()
	}

	if nodeAlloc.Cpu() != nil {
		allocs[v1.ResourceCPU] = *nodeAlloc.Cpu()
	}

	return
}

// calculatePodsTotalRequestsAndLimits calculates requests and limits of all the given pods
func calculatePodsTotalRequestsAndLimits(podList []v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, pod := range podList {
		podReqs, podLimits := resourceHelper.PodRequestsAndLimits(&pod)
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = *podReqValue.Copy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = *podLimitValue.Copy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}
	return
}

// InstallSecretsToCluster add all secrets from a repo to a cluster's namespace combined into one global secret named as the repo
func InstallSecretsToCluster(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	// bind request body to UpdateClusterRequest struct
	var request pkgSecret.InstallSecretsToClusterRequest
	if err := c.BindJSON(&request); err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	secretSources, err := cluster.InstallSecrets(commonCluster, &request.Query, request.Namespace)

	if err != nil {
		log.Errorf("Error installing secrets [%v] into cluster [%d]: %s", request.Query, commonCluster.GetID(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error installing secrets into cluster",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, secretSources)
}

// ProxyToCluster sets up a proxy and forwards all requests to the cluster's API server.
func (a *ClusterAPI) ProxyToCluster(c *gin.Context) {

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	apiProxyPrefix := strings.TrimSuffix(c.Request.URL.Path, c.Param("path"))

	kubeProxy, err := a.clusterManager.GetKubeProxy(apiProxyPrefix, commonCluster)
	if err != nil {
		log.Errorf("Error proxying to cluster [%d]: %s", commonCluster.GetID(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error proxying to cluster",
			Error:   err.Error(),
		})
		return
	}

	kubeProxy.Handler(c)
}

// ListClusterSecrets returns
func ListClusterSecrets(c *gin.Context) {

	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	releaseName := c.Query("releaseName")
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	log := log.WithFields(logrus.Fields{
		"organization": organizationID,
		"clusterId":    commonCluster.GetID(),
		"releaseName":  releaseName,
	})

	log.Info("Start filtering secrets")

	clusterUidTag := fmt.Sprintf("clusterUID:%s", commonCluster.GetUID())
	releaseTag := fmt.Sprintf("release:%s", releaseName)

	tags := []string{clusterUidTag}
	if len(releaseName) != 0 {
		tags = append(tags, releaseTag)
	}

	log.Infof("tags: %v", tags)

	secrets, err := secret.RestrictedStore.List(organizationID, &pkgSecret.ListSecretsQuery{
		Tags: tags,
	})
	if err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing secrets",
			Error:   err.Error(),
		})
		return
	}

	log.Info("Listing secrets succeeded")

	c.JSON(http.StatusOK, secrets)

}
