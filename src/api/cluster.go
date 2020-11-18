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

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
	v1 "k8s.io/api/core/v1"

	clusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	eksdriver "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/driver"
	"github.com/banzaicloud/pipeline/internal/cluster/resourcesummary"
	"github.com/banzaicloud/pipeline/internal/global"
	azureDriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	vsphereDriver "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver"
	"github.com/banzaicloud/pipeline/internal/secret/restricted"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/api/common"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

// ClusterAPI implements the Cluster API actions.
type ClusterAPI struct {
	clusterManager          *cluster.Manager
	clusterGetter           common.ClusterGetter
	externalBaseURL         string
	externalBaseURLInsecure bool
	workflowClient          client.Client
	clientFactory           common.DynamicClientFactory

	logger          logrus.FieldLogger
	errorHandler    emperror.Handler
	clusterCreators ClusterCreators
	clusterUpdaters ClusterUpdaters

	helmService        cluster.HelmService
	authConfig         auth.Config
	clientSecretGetter clusterAuth.ClusterClientSecretGetter
}

type ClusterCreators struct {
	PKEOnAzure   azureDriver.ClusterCreator
	EKSAmazon    eksdriver.EksClusterCreator
	PKEOnVsphere vsphereDriver.VspherePKEClusterCreator
}

type ClusterDeleters struct {
	PKEOnAzure azureDriver.ClusterDeleter
	EKSAmazon  eksdriver.EKSClusterDeleter
}

type ClusterUpdaters struct {
	PKEOnAzure   azureDriver.ClusterUpdater
	EKSAmazon    eksdriver.EksClusterUpdater
	PKEOnVsphere vsphereDriver.ClusterUpdater
}

// NewClusterAPI returns a new ClusterAPI instance.
func NewClusterAPI(
	clusterManager *cluster.Manager,
	clusterGetter common.ClusterGetter,
	workflowClient client.Client,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	externalBaseURL string,
	externalBaseURLInsecure bool,
	clusterCreators ClusterCreators,
	clusterUpdaters ClusterUpdaters,
	clientFactory common.DynamicClientFactory,
	helmService cluster.HelmService,
	authConfig auth.Config,
	clientSecretGetter clusterAuth.ClusterClientSecretGetter,
) *ClusterAPI {
	return &ClusterAPI{
		clusterManager:          clusterManager,
		clusterGetter:           clusterGetter,
		workflowClient:          workflowClient,
		externalBaseURL:         externalBaseURL,
		externalBaseURLInsecure: externalBaseURLInsecure,
		logger:                  logger,
		errorHandler:            errorHandler,
		clusterCreators:         clusterCreators,
		clusterUpdaters:         clusterUpdaters,
		clientFactory:           clientFactory,
		helmService:             helmService,
		authConfig:              authConfig,
		clientSecretGetter:      clientSecretGetter,
	}
}

// getClusterFromRequest just a simple getter to build commonCluster object this handles error messages directly
// Deprecated: use internal.clusterGetter instead
func getClusterFromRequest(c *gin.Context) (cluster.CommonCluster, bool) {
	// TODO: move these to a struct and create them only once upon application init
	clusters := clusteradapter.NewClusters(global.DB())
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterStore := clusteradapter.NewStore(global.DB(), clusters)
	clusterManager := cluster.NewManager(clusters, secretValidator, cluster.NewNopClusterEvents(), nil, nil, nil, log, errorHandler, clusterStore, nil)
	clusterGetter := common.NewClusterGetter(clusterManager, log, errorHandler)

	return clusterGetter.GetClusterFromRequest(c)
}

// GetClusterConfig gets a cluster config
func GetClusterConfig(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}
	config, err := commonCluster.GetK8sUserConfig()
	if err != nil {
		log.Debugf("error during getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting config",
			Error:   err.Error(),
		})
		return
	}

	cleanKubeConfig, err := k8sclient.CleanKubeconfig(config)
	if err != nil {
		log.Debugf("error during getting config: %s", err.Error())
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
			Data:   string(cleanKubeConfig),
		})
	default:
		c.String(http.StatusOK, string(cleanKubeConfig))
	}
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

	response, err := describePods(c.Request.Context(), commonCluster)
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

func describePods(ctx context.Context, commonCluster cluster.CommonCluster) (items []pkgCluster.PodDetailsResponse, err error) {
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
	pods, err = listPods(ctx, client, "", "")
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

// InstallSecretsToClusterRequest describes an InstallSecretToCluster request
type InstallSecretsToClusterRequest struct {
	Namespace string                  `json:"namespace" binding:"required"`
	Query     secret.ListSecretsQuery `json:"query" binding:"required"`
}

// InstallSecretsToCluster add all secrets from a repo to a cluster's namespace combined into one global secret named as the repo
func InstallSecretsToCluster(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if !ok {
		return
	}

	var request InstallSecretsToClusterRequest
	if err := c.BindJSON(&request); err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	secretSources, err := cluster.InstallSecrets(c.Request.Context(), commonCluster, &request.Query, request.Namespace)
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

	var query secret.ListSecretsQuery
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

	secrets, err := restricted.GlobalSecretStore.List(organizationID, &query)
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

	logger := a.logger.WithFields(logrus.Fields{
		"organization": cluster.GetOrganizationId(),
		"clusterName":  cluster.GetName(),
		"clusterID":    cluster.GetID(),
	})

	keys := []interface{}{
		"organization", cluster.GetOrganizationId(),
		"clusterName", cluster.GetName(),
		"clusterID", cluster.GetID(),
	}

	clusterGetCAHasher, ok := cluster.(interface {
		GetCAHash() (string, error)
	})
	if !ok {
		err := errors.New(fmt.Sprintf("not implemented for this type of cluster (%T)", cluster))
		a.errorHandler.Handle(errors.WithDetails(err, keys...))

		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Not implemented",
			Error:   err.Error(),
		})
		return
	}
	hash, err := clusterGetCAHasher.GetCAHash()
	if err != nil {
		message := "Kubernetes CA certificate (Kubeconfig) is not available yet"
		a.errorHandler.Handle(errors.WrapIfWithDetails(err, message, keys...))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	masterAddress, err := cluster.GetAPIEndpoint()
	if err != nil {
		message := "Error fetching kubernetes API address"
		logger.Info(errors.WrapIf(err, message))

		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   err.Error(),
		})
		return
	}
	url, err := url.Parse(masterAddress)
	if err != nil {
		message := "Error parsing kubernetes API address"
		a.errorHandler.Handle(errors.WrapIfWithDetails(err, message, keys...))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   err.Error(),
		})
		return
	}
	config, err := cluster.GetK8sConfig()
	if err != nil {
		message := "Error fetching Kubernetes config"
		logger.Info(errors.WrapIf(err, message))

		c.AbortWithStatusJSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   err.Error(),
		})
		return
	}
	client, err := k8sclient.NewClientFromKubeConfig(config)
	if err != nil {
		message := "Invalid Kubernetes config"
		a.errorHandler.Handle(errors.WrapIfWithDetails(err, message, keys...))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   err.Error(),
		})
		return
	}
	// Get an active token
	token, err := k8sutil.GetOrCreateBootstrapToken(c.Request.Context(), log, client)
	if err != nil {
		message := "Failed to create bootstrap token"
		logger.Info(errors.WrapIf(err, message))

		c.AbortWithStatusJSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
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
