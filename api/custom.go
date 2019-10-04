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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	"github.com/banzaicloud/pipeline/internal/common"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

type EndpointLister struct {
	logger common.Logger
}

func MakeEndpointLister(logger common.Logger) EndpointLister {
	return EndpointLister{
		logger: logger,
	}
}

// ListEndpoints lists service public endpoints
func (el EndpointLister) ListEndpoints(c *gin.Context) {

	releaseName := c.Query("releaseName")
	log.Infof("Filtering for helm release name: %s", releaseName)
	log.Info("if empty(\"\") all the endpoints will be returned")

	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}
	if releaseName != "" {
		status, err := helm.GetDeploymentStatus(releaseName, kubeConfig)
		if err != nil {
			c.JSON(int(status), pkgCommon.ErrorResponse{
				Code:    int(status),
				Message: err.Error(),
				Error:   err.Error(),
			})
			return
		}
	}

	logger := el.logger.WithContext(c)
	endpointManager := endpoints.NewEndpointManager(logger)
	endpointList, err := endpointManager.List(kubeConfig, releaseName)
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

//GetClusterNodes Get node information
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

	response, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
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
