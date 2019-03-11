// Copyright © 2019 Banzai Cloud
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// ReRunPostHooks reruns cluster posthooks.
func (a *ClusterAPI) ReRunPostHooks(c *gin.Context) {
	commonCluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if ok != true {
		return
	}

	logger := a.logger.WithField("clusterID", commonCluster.GetID())

	var ph pkgCluster.PostHooks
	if err := c.BindJSON(&ph); err != nil {
		logger.Debugf("cannot parse request: %s", err.Error())

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "cannot parse request",
			Error:   err.Error(),
		})

		return
	}

	logger.WithField("workflowName", cluster.RunPostHooksWorkflowName).Info("starting workflow")

	input := cluster.RunPostHooksWorkflowInput{
		ClusterID: commonCluster.GetID(),
		PostHooks: cluster.BuildWorkflowPostHookFunctions(ph, false),
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 2 * time.Hour, // TODO: lower timeout
	}

	exec, err := a.workflowClient.ExecuteWorkflow(c.Request.Context(), workflowOptions, cluster.RunPostHooksWorkflowName, input)
	if err != nil {
		a.errorHandler.Handle(emperror.WrapWith(err, "failed to start workflow", "workflowName", cluster.RunPostHooksWorkflowName))

		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to run posthooks",
		})

		return
	}

	logger.WithFields(logrus.Fields{
		"workflowName":  cluster.RunPostHooksWorkflowName,
		"workflowID":    exec.GetID(),
		"workflowRunID": exec.GetRunID(),
	}).Info("workflow started successfully")

	c.Status(http.StatusOK)
}
