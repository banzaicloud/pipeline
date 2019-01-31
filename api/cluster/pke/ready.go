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

package pke

import (
	"encoding/base64"
	"net/http"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

type ReadyRequest struct {
	Config   string `json:"config,omitempty"` // kubeconfig in base64 or empty if not a master
	Name     string `json:"name",required"`   // name of node
	NodePool string `json:"nodePool"`         // name of nodepool the new node belongs to
	IP       string `json:"ip,omitempty"`     // ip address of node (where the other nodes can reach it)
}

func (a *API) PostReady(c *gin.Context) {

	commonCluster, log, ok := a.getCluster(c)
	if !ok {
		err := errors.New("cluster not found")
		a.errorHandler.Handle(err)

		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Cluster not found",
			Error:   err.Error(),
		})
		return
	}

	var request ReadyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		err := emperror.Wrap(err, "could not parse request")
		a.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request",
			Error:   err.Error(),
		})
		return
	}

	log = log.WithField("node", request.Name)

	if request.Config != "" {
		decoded, err := base64.StdEncoding.DecodeString(request.Config)
		if err != nil {
			err := emperror.Wrap(err, "could not parse request")
			a.errorHandler.Handle(err)

			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Invalid base64 in config field",
				Error:   err.Error(),
			})
			return
		}

		if err := cluster.StoreKubernetesConfig(commonCluster, decoded); err != nil {
			err := emperror.Wrap(err, "could not store config")
			a.errorHandler.Handle(err)

			c.JSON(http.StatusInternalServerError, common.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to save config",
				Error:   err.Error(),
			})
			return
		}
		log.Info("Kubeconfig saved")
	}

	if registerNodeer, ok := commonCluster.(interface {
		RegisterNode(name, nodePool, ip string) error
	}); !ok {
		log.Infof("RegisterNode is not implemented in %T", commonCluster)
	} else {
		if err := registerNodeer.RegisterNode(request.Name, request.NodePool, request.IP); err != nil {
			err := emperror.Wrap(err, "could not store config")
			a.errorHandler.Handle(err)

			c.JSON(http.StatusInternalServerError, common.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to save config",
				Error:   err.Error(),
			})
			return
		}
		log.Info("node registered")
	}

	c.JSON(http.StatusOK, struct{}{})
}
