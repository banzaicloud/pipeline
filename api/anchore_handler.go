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

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	apiCommon "github.com/banzaicloud/pipeline/api/common"
	internalCommon "github.com/banzaicloud/pipeline/internal/common"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/pkg/common"
)

type PolicyHandler interface {
	ListPolicies(c *gin.Context)
	GetPolicy(c *gin.Context)
	CreatePolicy(c *gin.Context)
	DeletePolicy(c *gin.Context)
	UpdatePolicy(c *gin.Context)
}

type policyHandler struct {
	clusterGetter apiCommon.ClusterGetter
	policyService anchore.PolicyService
	logger        internalCommon.Logger
}

func NewPolicyHandler(clusterGetter apiCommon.ClusterGetter, policySvc anchore.PolicyService, logger internalCommon.Logger) PolicyHandler {
	return policyHandler{
		clusterGetter: clusterGetter,
		policyService: policySvc,
		logger:        logger.WithFields(map[string]interface{}{"policy-handler": "y"}),
	}
}

func (p policyHandler) ListPolicies(c *gin.Context) {

	cluster, ok := p.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		p.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	resp, err := p.policyService.ListPolicies(c.Request.Context(), cluster.GetOrganizationId(), cluster.GetID())
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to list policies",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (p policyHandler) GetPolicy(c *gin.Context) {
	policyId := c.Param("policyId")

	cluster, ok := p.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		p.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	resp, err := p.policyService.GetPolicy(c.Request.Context(), cluster.GetOrganizationId(), cluster.GetID(), policyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to get policy",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)

}

func (p policyHandler) CreatePolicy(c *gin.Context) {

	var policyBundle pipeline.PolicyBundleRecord
	if err := c.BindJSON(&policyBundle); err != nil {
		err := errors.WrapIf(err, "Error parsing request:")
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to bind the request",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	cluster, ok := p.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		p.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	resp, err := p.policyService.CreatePolicy(c.Request.Context(), cluster.GetOrganizationId(), cluster.GetID(), policyBundle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to get image meta info",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (p policyHandler) DeletePolicy(c *gin.Context) {

	policyId := c.Param("policyId")

	cluster, ok := p.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		p.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	if err := p.policyService.DeletePolicy(c.Request.Context(), cluster.GetOrganizationId(), cluster.GetID(), policyId); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to delete policy",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)

}

func (p policyHandler) UpdatePolicy(c *gin.Context) {

	policyId := c.Param("policyId")

	var policyBundleActivate pipeline.PolicyBundleActivate
	if err := c.BindJSON(&policyBundleActivate); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to bind the request body",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	cluster, ok := p.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		p.logger.Warn("failed to retrieve cluster based on the request")

		return
	}

	if err := p.policyService.UpdatePolicy(c.Request.Context(), cluster.GetOrganizationId(), cluster.GetID(),
		policyId, policyBundleActivate); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to update policy",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}
