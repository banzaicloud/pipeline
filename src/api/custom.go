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
	"strconv"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	"github.com/banzaicloud/pipeline/internal/common"
	intlHelm "github.com/banzaicloud/pipeline/internal/helm"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// Cluster collects operations to extract  cluster related information
type ClusterService interface {
	// Retrieves the kuebernetes configuration as a slice of bytes
	GetKubeConfig(ctx context.Context, clusterID uint) ([]byte, error)
}

type ReleaseChecker interface {
	CheckRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options intlHelm.Options) (string, error)
}

type EndpointLister struct {
	clusterService ClusterService
	releaseChecker ReleaseChecker
	logger         common.Logger
}

func MakeEndpointLister(clusterService ClusterService, releaseChecker ReleaseChecker, logger common.Logger) EndpointLister {
	return EndpointLister{
		clusterService: clusterService,
		releaseChecker: releaseChecker,
		logger:         logger,
	}
}

// ListEndpoints lists service public endpoints
func (el EndpointLister) ListEndpoints(c *gin.Context) {
	orgID, err := strconv.ParseUint(c.Param("orgid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	releaseName := c.Query("releaseName")
	log.Infof("Filtering for helm release name: %s", releaseName)
	log.Info("if empty(\"\") all the endpoints will be returned")

	if releaseName != "" {
		status, err := el.releaseChecker.CheckRelease(c.Request.Context(), uint(orgID), uint(clusterID), releaseName,
			intlHelm.Options{})
		if err != nil {
			c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Message: fmt.Sprintf("status: %s", status),
				Error:   err.Error(),
			})
			return
		}
	}

	kubeConfig, err := el.clusterService.GetKubeConfig(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Message: fmt.Sprintf("failed to retrieve k8s config for cluster: %d", clusterID),
			Error:   err.Error(),
		})
		return
	}

	logger := el.logger.WithContext(c)
	endpointManager := endpoints.NewEndpointManager(logger)
	endpointList, err := endpointManager.List(c.Request.Context(), kubeConfig, releaseName)
	if err != nil {
		var code int
		switch errors.Cause(err).(type) {
		case *endpoints.NotFoundError:
			code = http.StatusNotFound
		case *endpoints.PendingLoadBalancerError:
			code = http.StatusAccepted
		default:
			code = http.StatusInternalServerError
		}

		c.JSON(code, pkgCommon.ErrorResponse{
			Code:    code,
			Message: "Error during listing endpoints",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, pkgHelm.EndpointResponse{
		Endpoints: endpointList,
	})
}

// GetClusterNodes Get node information
func GetClusterNodes(c *gin.Context) {
	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting k8s connection: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting k8s connection",
			Error:   err.Error(),
		})

		return
	}

	response, err := client.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error listing nodes: %s", err.Error())
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Error during listing nodes",
			Error:   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, response)
}
