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

package pke

import (
	"net/http"

	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type leaderElectionRequest struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type leaderElectionResponse struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

// GetLeaderElection -
func (a *API) GetLeaderElection(c *gin.Context) {
	cluster, _, ok := a.getCluster(c)
	if !ok {
		return
	}

	leaderInfo, err := a.leaderRepository.GetLeader(cluster.GetOrganizationId(), cluster.GetID())
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to get leader",
			Error:   err.Error(),
		})
		return
	}

	if leaderInfo == nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	c.JSON(http.StatusOK, leaderElectionResponse{
		Hostname: leaderInfo.Hostname,
		IP:       leaderInfo.IP,
	})
	return
}

// DeleteLeaderElection -
func (a *API) DeleteLeaderElection(c *gin.Context) {
	cluster, _, ok := a.getCluster(c)
	if !ok {
		return
	}

	status := http.StatusOK

	err := a.leaderRepository.DeleteLeader(cluster.GetOrganizationId(), cluster.GetID())
	if err != nil {
		status = http.StatusInternalServerError
	}

	c.JSON(status, nil)
	return
}

// PostLeaderElection handles leader applications
func (a *API) PostLeaderElection(c *gin.Context) {
	cluster, _, ok := a.getCluster(c)
	if !ok {
		return
	}

	var req leaderElectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to parse request",
			Error:   err.Error(),
		})
		return
	}

	leaderInfo := &LeaderInfo{
		Hostname: req.Hostname,
		IP:       req.IP,
	}

	status := http.StatusCreated

	if err := a.leaderRepository.SetLeader(cluster.GetOrganizationId(), cluster.GetID(), *leaderInfo); err != nil {
		if !isLeaderSet(err) {
			ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to set leader",
				Error:   err.Error(),
			})
			return
		}

		status = http.StatusConflict
		leaderInfo, err = a.leaderRepository.GetLeader(cluster.GetOrganizationId(), cluster.GetID())
		if err != nil {
			ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to get leader",
				Error:   err.Error(),
			})
			return
		}
	}

	c.JSON(status, leaderElectionResponse{
		Hostname: leaderInfo.Hostname,
		IP:       leaderInfo.IP,
	})
	return
}

func isLeaderSet(err error) bool {
	type leaderSetter interface {
		LeaderSet() bool
	}

	err = errors.Cause(err)
	if e, ok := err.(leaderSetter); ok {
		return e.LeaderSet()
	}

	return false
}
