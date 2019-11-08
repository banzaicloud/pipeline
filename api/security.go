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
	"regexp"
	"strings"

	"emperror.dev/errors"
	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	clientV1alpha1 "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	apiCommon "github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/helm"
	internalCommon "github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/global"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"
)

func init() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
}

func getSecurityClient(c *gin.Context) *clientV1alpha1.SecurityV1Alpha1Client {
	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return nil
	}
	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting K8s config: %s", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return nil
	}

	securityClientSet, err := clientV1alpha1.SecurityConfig(config)
	if err != nil {
		log.Errorf("Error getting SecurityClient: %s", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting SecurityClient",
			Error:   err.Error(),
		})
		return nil
	}
	return securityClientSet
}

// GetImageDeployments list deployments by image
func GetImageDeployments(c *gin.Context) {
	imageDigest := c.Param("imageDigest")
	releaseMap := make(map[string]bool)

	re := regexp.MustCompile("^sha256:[a-f0-9]{64}$")
	if !re.MatchString(imageDigest) {
		err := fmt.Errorf("Invalid imageID format: %s", imageDigest)
		log.Error(err)
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return
	}

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return
	}

	// Get active helm deployments
	log.Info("Get deployments")
	activeReleases, err := helm.ListDeployments(nil, c.Query("tag"), kubeConfig)
	if err != nil {
		log.Error("Error listing deployments: ", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting K8s config: %s", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return
	}
	// Get all pods from cluster
	pods, err := listPods(client, "", "")
	if err != nil {
		log.Errorf("Error getting pods from cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting pods from cluster",
			Error:   err.Error(),
		})
		return
	}
	// Example status
	//	- containerID: docker://a8130dc313a40b0eb9151685ba41f84cd0e4bb7e2888c52691590ff8a22a2e6b
	//	image: banzaicloud/pipeline:0.4.0-dev29
	//	imageID: docker-pullable://banzaicloud/pipeline@sha256:5042ef1a5415dae8330583448584be2bb592053416b7db5fc41389a717cc52ab
	for _, p := range pods {
		for _, status := range p.Status.ContainerStatuses {
			if getImageDigest(status.ImageID) == imageDigest {
				releaseMap[pkgHelm.GetHelmReleaseName(p.Labels)] = true
			}
		}
		for _, status := range p.Status.InitContainerStatuses {
			if getImageDigest(status.ImageID) == imageDigest {
				releaseMap[pkgHelm.GetHelmReleaseName(p.Labels)] = true
			}
		}
	}
	releases := ListHelmReleases(c, activeReleases, releaseMap)

	c.JSON(http.StatusOK, releases)
}

func getImageDigest(imageID string) string {

	image := strings.Split(imageID, "@")
	if len(image) > 1 {
		return image[1]
	}
	return ""
}

// GetWhitelistSet will return a WhitelistSet
func GetWhitelistSet(c *gin.Context) (map[string]bool, bool) {
	securityClientSet := getSecurityClient(c)
	releaseWhitelist := make(map[string]bool)
	if securityClientSet == nil {
		return releaseWhitelist, false
	}
	whitelists, err := securityClientSet.Whitelists().List(metav1.ListOptions{})
	if err != nil {
		log.Warnf("can not fetch WhiteList: %s", err.Error())
		return releaseWhitelist, false
	}
	for _, whitelist := range whitelists.Items {
		releaseWhitelist[whitelist.ObjectMeta.Name] = true
	}
	log.Debugf("Whitelist set: %#v", releaseWhitelist)
	return releaseWhitelist, true
}

// GetReleaseScanLog will return a ReleaseScanlog
func GetReleaseScanLog(c *gin.Context) (map[string]bool, bool) {
	securityClientSet := getSecurityClient(c)
	releaseScanLogReject := make(map[string]bool)
	if securityClientSet == nil {
		return releaseScanLogReject, false
	}
	audits, err := securityClientSet.Audits().List(metav1.ListOptions{LabelSelector: "fakerelease=false"})
	if err != nil {
		log.Warnf("can not fetch ScanLog: %s", err.Error())
		return releaseScanLogReject, false
	}
	for _, audit := range audits.Items {
		if audit.Spec.Action == "reject" {
			releaseScanLogReject[audit.Spec.ReleaseName] = true
		}
	}
	log.Debugf("ReleaseScanLogReject set: %#v", releaseScanLogReject)
	return releaseScanLogReject, true
}

// SecurityScanEnabled checks if security scan is enabled in pipeline
func SecurityScanEnabled(c *gin.Context) {
	if global.Config.Cluster.SecurityScan.Anchore.Enabled {
		c.JSON(http.StatusOK, gin.H{
			"enabled": true,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled": false,
	})
	return
}

// SecurityHandler defines security related handler functions intended to be used for defining routes
type SecurityHandler interface {
	WhitelistHandler
	ScanLogHandler
}

type WhitelistHandler interface {
	GetWhiteLists(c *gin.Context)
	CreateWhiteList(c *gin.Context)
	DeleteWhiteList(c *gin.Context)
}

type ScanLogHandler interface {
	ListScanLogs(c *gin.Context)
	GetScanLogs(c *gin.Context)
}

type securityHandlers struct {
	clusterGetter   apiCommon.ClusterGetter
	resourceService anchore.SecurityResourceService
	errorHandler    internalCommon.ErrorHandler
	logger          internalCommon.Logger
}

func NewSecurityApiHandlers(
	clusterGetter apiCommon.ClusterGetter,
	errorHandler internalCommon.ErrorHandler,
	logger internalCommon.Logger) SecurityHandler {

	wlSvc := anchore.NewSecurityResourceService(logger)
	return securityHandlers{
		clusterGetter:   clusterGetter,
		resourceService: wlSvc,
		errorHandler:    errorHandler,
		logger:          logger,
	}
}

func (s securityHandlers) GetWhiteLists(c *gin.Context) {

	cluster, ok := s.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		s.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	whitelist, err := s.resourceService.GetWhitelists(c.Request.Context(), cluster)
	if err != nil {
		s.errorHandler.Handle(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error while retrieving whitelists",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	releaseWhitelist := make([]security.ReleaseWhiteListItem, 0)
	for _, whitelist := range whitelist {
		whitelistItem := security.ReleaseWhiteListItem{
			Name:   whitelist.Name,
			Owner:  whitelist.Spec.Creator,
			Reason: whitelist.Spec.Reason,
		}
		releaseWhitelist = append(releaseWhitelist, whitelistItem)
	}

	s.successResponse(c, releaseWhitelist)
}

func (s securityHandlers) CreateWhiteList(c *gin.Context) {
	cluster, ok := s.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		s.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	var whiteListItem *security.ReleaseWhiteListItem
	if err := c.BindJSON(&whiteListItem); err != nil {
		s.errorHandler.Handle(c.Request.Context(), err)

		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	if _, err := s.resourceService.CreateWhitelist(c.Request.Context(), cluster, *whiteListItem); err != nil {
		s.errorHandler.Handle(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error while creating whitelist",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.Status(http.StatusCreated)

}

func (s securityHandlers) DeleteWhiteList(c *gin.Context) {

	whitelisItemtName := c.Param("name")
	if whitelisItemtName == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "WhiteList name is required!",
			Error:   "WhiteList name is required!",
		})
		return
	}

	cluster, ok := s.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		s.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	if err := s.resourceService.DeleteWhitelist(c.Request.Context(), cluster, whitelisItemtName); err != nil {
		s.errorHandler.Handle(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error while deleting whitelist",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

func (s securityHandlers) ListScanLogs(c *gin.Context) {
	cluster, ok := s.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		s.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	scanlogs, err := s.resourceService.ListScanLogs(c.Request.Context(), cluster)
	if err != nil {
		s.errorHandler.Handle(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to list scan logs",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusOK, scanlogs)
}

func (s securityHandlers) GetScanLogs(c *gin.Context) {
	releaseName := c.Param("releaseName")

	cluster, ok := s.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		s.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	scanlogs, err := s.resourceService.GetScanLogs(c.Request.Context(), cluster, releaseName)
	if err != nil {
		s.errorHandler.Handle(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to retrieve scan logs",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusOK, scanlogs)
}

func (s securityHandlers) successResponse(ginCtx *gin.Context, payload interface{}) {
	ginCtx.JSON(http.StatusOK, payload)
	return
}
