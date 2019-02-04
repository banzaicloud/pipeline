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
	"fmt"
	"net/http"

	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// ListCommands Lists the commands for installing nodes in each nodePool
func (a *API) ListCommands(c *gin.Context) {
	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
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
	clusterCommander, ok := cluster.(interface {
		GetBootstrapCommand(nodePool, url, token, clusterName string) string
		GetPipelineToken(tokenGenerator interface{}) (string, error)
	})
	if !ok {
		err := errors.New(fmt.Sprintf("not implemented for this type of cluster (%T)", cluster))
		a.errorHandler.Handle(err)

		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Not implemented",
			Error:   err.Error(),
		})
		return
	}

	commands := map[string]string{}

	nodePools, err := cluster.ListNodeNames()
	if err != nil {
		err := emperror.Wrap(err, "can't list nodes")
		a.errorHandler.Handle(err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
			Error:   err.Error(),
		})
		return
	}

	token, err := clusterCommander.GetPipelineToken(a.tokenGenerator)
	if err != nil {
		err := emperror.Wrap(err, "can't generate token")
		a.errorHandler.Handle(err)

		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
			Error:   err.Error(),
		})
		return
	}

	for nodePool := range nodePools {
		commands[nodePool] = clusterCommander.GetBootstrapCommand(nodePool, a.externalBaseURL, token, cluster.GetName())
	}

	if len(commands) == 0 { // give some examples for the user...
		commands["master"] = clusterCommander.GetBootstrapCommand("master", a.externalBaseURL, token, cluster.GetName())
		commands["pool1"] = clusterCommander.GetBootstrapCommand("pool1", a.externalBaseURL, token, cluster.GetName())
	}

	c.JSON(http.StatusOK, commands)
}
