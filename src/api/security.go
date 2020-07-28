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
	"regexp"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	internalCommon "github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"
	apiCommon "github.com/banzaicloud/pipeline/src/api/common"
)

func init() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
}

func getClusterClient(c *gin.Context) client.Client {
	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return nil
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		log.Errorf("failed to create k8s client config for cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to create k8s client config",
			Error:   err.Error(),
		})
		return nil
	}

	cli, err := client.New(config, client.Options{})
	if err != nil {
		log.Errorf("failed to create k8s client for cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to create k8s client for cluster",
			Error:   err.Error(),
		})
		return nil
	}

	return cli
}

// ReleaseLister helm operation abstraction interface
type ReleaseLister interface {
	// ListReleases lists helm releases for the given input parameters
	ListReleases(ctx context.Context, organizationID uint, clusterID uint, releaseFilter helm.ReleaseFilter, options helm.Options) ([]helm.Release, error)
}

// imageDeploymentsHandler providing helm abstraction to the handler
type imageDeploymentsHandler struct {
	clusterService ClusterService
	releaseLister  ReleaseLister
	logger         internalCommon.Logger
}

func NewImageDeploymentsHandler(releaseLister ReleaseLister, clusterService ClusterService, logger internalCommon.Logger) imageDeploymentsHandler {
	return imageDeploymentsHandler{
		releaseLister:  releaseLister,
		clusterService: clusterService,
		logger:         logger,
	}
}

func (i imageDeploymentsHandler) GetImageDeployments(c *gin.Context) {
	orgID, err := strconv.ParseUint(c.Param("orgid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to get path param",
			Error:   err.Error(),
		})
		return
	}

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to get path param",
			Error:   err.Error(),
		})
		return
	}

	imageDigest := c.Param("imageDigest")
	re := regexp.MustCompile("^sha256:[a-f0-9]{64}$")
	if !re.MatchString(imageDigest) {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid image digest format",
			Error:   fmt.Sprintf("invalid imageID format: %s", imageDigest),
		})
		return
	}

	kubeConfig, err := i.clusterService.GetKubeConfig(c.Request.Context(), uint(clusterID))
	if err != nil {
		i.logger.Error("failed to retrieve kubernetes configuration for cluster", map[string]interface{}{"clusterID": clusterID, "messageDigesr": imageDigest})
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to retrieve kubernetes configuration for cluster",
			Error:   err.Error(),
		})
		return
	}

	activeReleases, err := i.releaseLister.ListReleases(c.Request.Context(), uint(orgID), uint(clusterID),
		helm.ReleaseFilter{TagFilter: c.Query("tag")}, helm.Options{})
	if err != nil {
		i.logger.Error("failed to list releases", map[string]interface{}{"clusterID": clusterID, "messageDigesr": imageDigest})
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting K8s config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting K8s config",
			Error:   err.Error(),
		})
		return
	}
	// Get all pods from cluster
	pods, err := listPods(c.Request.Context(), client, "", "")
	if err != nil {
		log.Errorf("Error getting pods from cluster: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
	releaseMap := make(map[string]bool)
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
	cli := getClusterClient(c)

	releaseWhitelist := make(map[string]bool)
	whitelists := &v1alpha1.WhiteListItemList{}

	if err := cli.List(c, whitelists, &client.ListOptions{}); err != nil {
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
	cli := getClusterClient(c)

	releaseScanLogReject := make(map[string]bool)
	audits := &v1alpha1.AuditList{}

	if err := cli.List(c, audits, client.MatchingLabels(map[string]string{"fakerelease": "false"})); err != nil {
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
		s.errorHandler.HandleContext(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
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
		s.errorHandler.HandleContext(c.Request.Context(), err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	if _, err := s.resourceService.CreateWhitelist(c.Request.Context(), cluster, *whiteListItem); err != nil {
		s.errorHandler.HandleContext(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
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
		s.errorHandler.HandleContext(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
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
		s.errorHandler.HandleContext(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
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
		s.errorHandler.HandleContext(c.Request.Context(), err)

		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
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
