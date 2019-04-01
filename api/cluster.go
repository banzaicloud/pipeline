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
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence/client"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cloudinfo"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/resourcesummary"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/pkg/providers"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

// ClusterAPI implements the Cluster API actions.
type ClusterAPI struct {
	clusterManager  *cluster.Manager
	clusterGetter   common.ClusterGetter
	externalBaseURL string
	workflowClient  client.Client

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewClusterAPI returns a new ClusterAPI instance.
func NewClusterAPI(
	clusterManager *cluster.Manager,
	clusterGetter common.ClusterGetter,
	workflowClient client.Client,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	externalBaseURL string,

) *ClusterAPI {
	return &ClusterAPI{
		clusterManager:  clusterManager,
		clusterGetter:   clusterGetter,
		workflowClient:  workflowClient,
		externalBaseURL: externalBaseURL,

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
	clusterManager := cluster.NewManager(clusters, secretValidator, cluster.NewNopClusterEvents(), nil, nil, nil, log, errorHandler)
	clusterGetter := common.NewClusterGetter(clusterManager, log, errorHandler)

	return clusterGetter.GetClusterFromRequest(c)
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
			// TODO we want skip or return error?
			logger.Errorf("get cluster status failed: %s", err.Error())
		} else {
			response = append(response, *status)
		}
	}

	c.JSON(http.StatusOK, response)
}

// ClusterCheck checks the cluster ready
func (a *ClusterAPI) ClusterCheck(c *gin.Context) {
	commonCluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if ok != true {
		return
	}

	ok, err := commonCluster.IsReady()
	if err != nil {
		errorHandler.Handle(err)

		c.Status(http.StatusInternalServerError)
		return
	}

	if !ok {
		c.Status(http.StatusNotFound)
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
		req, limits := resourcesummary.CalculatePodsTotalRequestsAndLimits([]v1.Pod{pod})

		summary := resourcesummary.GetSummary(nil, nil, req, limits)

		items = append(items, pkgCluster.PodDetailsResponse{
			Name:          pod.Name,
			Namespace:     pod.Namespace,
			CreatedAt:     pod.CreationTimestamp.Time,
			Labels:        pod.Labels,
			RestartPolicy: string(pod.Spec.RestartPolicy),
			Conditions:    pod.Status.Conditions,
			Summary: &pkgCluster.ResourceSummary{
				Cpu: &pkgCluster.CPU{
					ResourceSummaryItem: pkgCluster.ResourceSummaryItem(summary.CPU),
				},
				Memory: &pkgCluster.Memory{
					ResourceSummaryItem: pkgCluster.ResourceSummaryItem(summary.Memory),
				},
			},
		})
	}

	return

}

// GetNodePools fetch node pool info for a cluster
func GetNodePools(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	clusterStatus, err := commonCluster.GetStatus()
	if err != nil {
		err = emperror.Wrap(err, "could not get cluster status")
		errorHandler.Handle(err)
		c.JSON(http.StatusServiceUnavailable, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "could not get cluster status",
			Error:   err.Error(),
		})
		return
	}

	nodePoolStatus := make(map[string]*pkgCluster.ActualNodePoolStatus)
	clusterDesiredResources := make(map[string]float64)
	clusterTotalResources := make(map[string]float64)

	autoScaleEnabled := commonCluster.GetScaleOptions() != nil && commonCluster.GetScaleOptions().Enabled
	if autoScaleEnabled {
		nodePoolCounts, err := getActualNodeCounts(commonCluster)
		if err != nil {
			err = emperror.Wrap(err, "could not get actual node count for node pool info")
			errorHandler.Handle(err)
			c.JSON(http.StatusServiceUnavailable, pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "could not get actual node count for node pool info",
				Error:   err.Error(),
			})
			return
		}

		headNodePoolName := viper.GetString(config.PipelineHeadNodePoolName)
		for nodePoolName, nodePool := range clusterStatus.NodePools {
			if nodePoolName == headNodePoolName {
				continue
			}

			nodePoolStatus[nodePoolName] = &pkgCluster.ActualNodePoolStatus{
				NodePoolStatus: *nodePool,
				ActualCount:    nodePoolCounts[nodePoolName],
			}

			machineDetails, err := cloudinfo.GetMachineDetails(log, clusterStatus.Cloud,
				clusterStatus.Distribution,
				clusterStatus.Region,
				nodePool.InstanceType)
			if err != nil {
				errorHandler.Handle(err)
			} else if machineDetails != nil {
				clusterTotalResources["cpu"] += float64(nodePool.Count) * machineDetails.CpusPerVm
				clusterTotalResources["gpu"] += float64(nodePool.Count) * machineDetails.GpusPerVm
				clusterTotalResources["mem"] += float64(nodePool.Count) * machineDetails.MemPerVm
			}
		}

		clusterDesiredResources["cpu"] += commonCluster.GetScaleOptions().DesiredCpu
		clusterDesiredResources["gpu"] += float64(commonCluster.GetScaleOptions().DesiredGpu)
		clusterDesiredResources["mem"] += commonCluster.GetScaleOptions().DesiredMem
		clusterDesiredResources["onDemandPct"] += float64(commonCluster.GetScaleOptions().OnDemandPct)
	}

	response := pkgCluster.GetNodePoolsResponse{
		ScaleEnabled:            autoScaleEnabled,
		NodePools:               nodePoolStatus,
		ClusterDesiredResources: clusterDesiredResources,
		ClusterTotalResources:   clusterTotalResources,
		ClusterStatus:           clusterStatus.Status,
		Cloud:                   clusterStatus.Cloud,
		Distribution:            clusterStatus.Distribution,
		Location:                clusterStatus.Location,
	}

	c.JSON(http.StatusOK, response)
}

func getActualNodeCounts(commonCluster cluster.CommonCluster) (map[string]int, error) {
	nodePoolCounts := make(map[string]int)
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create new k8s client")
	}

	nodes, err := client.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, emperror.Wrap(err, "could not get nodes list from cluster")
	}

nodesloop:
	for _, node := range nodes.Items {
		// don't count cordoned nodes (Unschedulable and tainted with node.banzaicloud.io/draining)
		if node.Spec.Unschedulable {
			continue
		}
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node.banzaicloud.io/draining" {
				continue nodesloop
			}
		}
		nodePoolName := node.Labels[pkgCommon.LabelKey]
		if len(nodePoolName) > 0 {
			nodePoolCounts[nodePoolName] += 1
		}
	}

	return nodePoolCounts, nil
}

// InstallSecretsToCluster add all secrets from a repo to a cluster's namespace combined into one global secret named as the repo
func InstallSecretsToCluster(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

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

	kubeProxy, err := a.clusterManager.GetKubeProxy(c.Request.URL.Scheme, c.Request.URL.Host, apiProxyPrefix, commonCluster)
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

	var query pkgSecret.ListSecretsQuery
	err := c.BindQuery(&query)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Failed to parse query",
			Error:   err.Error(),
		})
		return
	}

	log.Debugln("secret query ", "type:", query.Type, "tags:", query.Tags, "values:", query.Values)

	clusterUidTag := fmt.Sprintf("clusterUID:%s", commonCluster.GetUID())
	releaseTag := fmt.Sprintf("release:%s", releaseName)

	query.Tags = append(query.Tags, clusterUidTag)
	if len(releaseName) != 0 {
		query.Tags = append(query.Tags, releaseTag)
	}

	secrets, err := secret.RestrictedStore.List(organizationID, &query)
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

type clusterBootstrapInfo struct {
	Token                    string `json:"token"`
	DiscoveryTokenCaCertHash string `json:"discoveryTokenCaCertHash"`
	MasterAddress            string `json:"masterAddress"`
}

// GetBootstrapInfo
func (a *ClusterAPI) GetBootstrapInfo(c *gin.Context) {
	// Fetch cluster information
	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}
	clusterGetCAHasher, ok := cluster.(interface {
		GetCAHash() (string, error)
	})
	if !ok {
		err := errors.New(fmt.Sprintf("not implemented for this type of cluster (%T)", cluster))
		a.errorHandler.Handle(err)

		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Not implemented",
			Error:   err.Error(),
		})
		return
	}
	hash, err := clusterGetCAHasher.GetCAHash()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Kubernetes CA certificate (Kubeconfig) is not available yet",
			Error:   err.Error(),
		})
		return
	}

	masterAddress, err := cluster.GetAPIEndpoint()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error fetching kubernetes API address",
			Error:   err.Error(),
		})
		return
	}
	url, err := url.Parse(masterAddress)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing kubernetes API address",
			Error:   err.Error(),
		})
		return
	}
	config, err := cluster.GetK8sConfig()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error fetching Kubernetes config",
			Error:   err.Error(),
		})
		return
	}
	client, err := k8sclient.NewClientFromKubeConfig(config)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid Kubernetes config",
			Error:   err.Error(),
		})
		return
	}
	// Get an active token
	token, err := k8sutil.GetOrCreateBootstrapToken(log, client)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Failed to create bootstrap token",
			Error:   err.Error(),
		})
		return
	}
	bootstrapInfo := &clusterBootstrapInfo{
		Token:                    token,
		DiscoveryTokenCaCertHash: hash,
		MasterAddress:            url.Host,
	}
	c.JSON(http.StatusOK, bootstrapInfo)
}
