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

	"github.com/mitchellh/mapstructure"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	clusterAPI "github.com/banzaicloud/pipeline/src/api/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

func decodeRequest(input map[string]interface{}, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  output,
		TagName: "json",
	})

	if err != nil {
		return err
	}

	return decoder.Decode(input)
}

func (a *ClusterAPI) parseRequest(ctx *gin.Context, body map[string]interface{}, req interface{}) bool {
	if err := decodeRequest(body, req); err != nil {
		err = errors.WrapIff(err, "failed to parse request into %T", req)

		a.errorHandler.Handle(err)
		pkgCommon.ErrorResponseWithStatus(ctx, http.StatusBadRequest, err)

		return false
	}
	return true
}

func isInputValidationError(err error) bool {
	var e interface {
		InputValidationError() bool
	}

	return errors.As(err, &e) && e.InputValidationError()
}

func (a *ClusterAPI) handleCreationError(ctx *gin.Context, err error) {
	a.errorHandler.Handle(err)

	status := http.StatusInternalServerError
	if isInputValidationError(err) || isInvalid(err) {
		status = http.StatusBadRequest
	}
	pkgCommon.ErrorResponseWithStatus(ctx, status, err)
}

// CreateCluster creates a K8S cluster in the cloud.
func (a *ClusterAPI) CreateCluster(c *gin.Context) {
	a.logger.Info("Cluster creation started")

	ctx := ginutils.Context(context.Background(), c)

	orgID := auth.GetCurrentOrganization(c.Request).ID
	userID := auth.GetCurrentUser(c.Request).ID

	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		a.errorHandler.Handle(err)
		pkgCommon.ErrorResponseWithStatus(c, http.StatusBadRequest, err)
		return
	}
	if _, ok := requestBody["type"]; !ok {
		a.logger.Info("request body did not match v2 structure, trying legacy path")
		var createClusterRequest pkgCluster.CreateClusterRequest
		if !a.parseRequest(c, requestBody, &createClusterRequest) {
			return
		}

		if createClusterRequest.SecretId == "" && len(createClusterRequest.SecretIds) == 0 {
			if createClusterRequest.SecretName == "" {
				c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
					Code:    http.StatusBadRequest,
					Message: "either secretId or secretName has to be set",
				})
				return
			}

			createClusterRequest.SecretId = secret.GenerateSecretIDFromName(createClusterRequest.SecretName)
		}

		commonCluster, err := a.createCluster(ctx, &createClusterRequest, orgID, userID, createClusterRequest.PostHooks)
		if err != nil {
			c.JSON(err.Code, err)
			return
		}

		c.JSON(http.StatusAccepted, pkgCluster.CreateClusterResponse{
			Name:       commonCluster.GetName(),
			ResourceID: commonCluster.GetID(),
		})
		return
	}

	var createClusterRequestBase pipeline.CreateClusterRequestBase
	if !a.parseRequest(c, requestBody, &createClusterRequestBase) {
		return
	}

	secretID := createClusterRequestBase.SecretId
	if secretID == "" {
		if createClusterRequestBase.SecretName != "" {
			secretID = secret.GenerateSecretIDFromName(createClusterRequestBase.SecretName)
		} else {
			ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "either secret ID or name is required",
				Error:   "no secret specified",
			})
			return
		}
	}

	var cluster interface {
		GetID() uint
		GetName() string
	}

	switch createClusterRequestBase.Type {
	case clusterAPI.PKEOnVsphere:
		var req clusterAPI.CreatePKEOnVsphereClusterRequest
		if ok := a.parseRequest(c, requestBody, &req); !ok {
			return
		}
		req.SecretId = secretID
		// TODO legacy posthook support if needed
		params := req.ToVspherePKEClusterCreationParams(orgID, userID)
		a.logger.Infof("request: %+v\n\n\nparams: %+v\n\n", req, params)
		vsphereCluster, err := a.clusterCreators.PKEOnVsphere.Create(ctx, params)
		if err = errors.WrapIf(err, "failed to create cluster from request"); err != nil {
			a.handleCreationError(c, err)
			return
		}
		cluster = vsphereCluster
	case clusterAPI.PKEOnAzure:
		var req clusterAPI.CreatePKEOnAzureClusterRequest
		if ok := a.parseRequest(c, requestBody, &req); !ok {
			return
		}
		req.SecretId = secretID
		params := req.ToAzurePKEClusterCreationParams(orgID, userID)
		azurePKECluster, err := a.clusterCreators.PKEOnAzure.Create(ctx, params)
		if err = errors.WrapIf(err, "failed to create cluster from request"); err != nil {
			a.handleCreationError(c, err)
			return
		}
		cluster = azurePKECluster
	default:
		ginutils.ReplyWithErrorResponse(c, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("unknown cluster type: %s", createClusterRequestBase.Type),
		})
		return
	}

	c.JSON(http.StatusAccepted, pkgCluster.CreateClusterResponse{
		Name:       cluster.GetName(),
		ResourceID: cluster.GetID(),
	})
}

// createCluster creates a K8S cluster in the cloud.
func (a *ClusterAPI) createCluster(
	ctx context.Context,
	createClusterRequest *pkgCluster.CreateClusterRequest,
	organizationID uint,
	userID uint,
	postHooks pkgCluster.PostHooks,
) (cluster.CommonCluster, *pkgCommon.ErrorResponse) {
	logger := a.logger.WithFields(logrus.Fields{
		"organization": organizationID,
		"user":         userID,
		"cluster":      createClusterRequest.Name,
	})

	logger.Infof("Creating new entry with cloud type: %s", createClusterRequest.Cloud)

	// TODO check validation
	// This is the common part of cluster flow
	commonCluster, err := cluster.CreateCommonClusterFromRequest(createClusterRequest, organizationID, userID)
	if err != nil {
		log.Errorf("error during create common cluster from request: %s", err.Error())
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		}
	}

	if _, ok := commonCluster.(*cluster.EKSCluster); ok {
		commonCluster, err = a.clusterCreators.EKSAmazon.CreateCluster(ctx, commonCluster, createClusterRequest, organizationID, userID)
	} else {
		creationCtx := cluster.CreationContext{
			OrganizationID:          organizationID,
			UserID:                  userID,
			Name:                    createClusterRequest.Name,
			SecretID:                createClusterRequest.SecretId,
			SecretIDs:               createClusterRequest.SecretIds,
			Provider:                createClusterRequest.Cloud,
			PostHooks:               postHooks,
			ExternalBaseURL:         a.externalBaseURL,
			ExternalBaseURLInsecure: a.externalBaseURLInsecure,
		}
		creator := cluster.NewClusterCreator(createClusterRequest, commonCluster, a.workflowClient)
		commonCluster, err = a.clusterManager.CreateCluster(ctx, creationCtx, creator)
	}

	if err == cluster.ErrAlreadyExists || isInvalid(err) {
		logger.Debugf("invalid cluster creation: %s", err.Error())

		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		}
	} else if err != nil {
		logger.Errorf("error during cluster creation: %s", err.Error())

		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
			Error:   err.Error(),
		}
	}

	return commonCluster, nil
}
