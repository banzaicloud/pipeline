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
	"github.com/banzaicloud/pipeline/internal/cluster/resourcesummary"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/providers"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

// ClusterAPI implements the Cluster API actions.
type ClusterAPI struct {
	clusterManager *cluster.Manager
	clusterGetter  common.ClusterGetter

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewClusterAPI returns a new ClusterAPI instance.
func NewClusterAPI(clusterManager *cluster.Manager, clusterGetter common.ClusterGetter, logger logrus.FieldLogger, errorHandler emperror.Handler) *ClusterAPI {
	return &ClusterAPI{
		clusterManager: clusterManager,
		clusterGetter:  clusterGetter,

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
