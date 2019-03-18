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
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//CreateClusterRequest gin handler
func (a *ClusterAPI) CreateClusterRequest(c *gin.Context) {
	a.logger.Info("Cluster creation started")

	a.logger.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createClusterRequest pkgCluster.CreateClusterRequest
	if err := c.BindJSON(&createClusterRequest); err != nil {
		a.logger.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
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

		createClusterRequest.SecretId = string(secret.GenerateSecretIDFromName(createClusterRequest.SecretName))
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	userID := auth.GetCurrentUser(c.Request).ID

	ctx := ginutils.Context(context.Background(), c)
	commonCluster, err := a.CreateCluster(ctx, &createClusterRequest, orgID, userID, createClusterRequest.PostHooks)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusAccepted, pkgCluster.CreateClusterResponse{
		Name:       commonCluster.GetName(),
		ResourceID: commonCluster.GetID(),
	})
}

// CreateCluster creates a K8S cluster in the cloud
func (a *ClusterAPI) CreateCluster(
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

	// TODO: refactor profile handling as well?
	if len(createClusterRequest.ProfileName) != 0 {
		logger = logger.WithField("profile", createClusterRequest.ProfileName)

		logger.Info("fill data from profile")

		distribution := pkgCluster.Unknown
		switch createClusterRequest.Cloud {
		case pkgCluster.Amazon:
			distribution = pkgCluster.EKS
		case pkgCluster.Azure:
			distribution = pkgCluster.AKS
		case pkgCluster.Google:
			distribution = pkgCluster.GKE
		case pkgCluster.Oracle:
			distribution = pkgCluster.OKE
		default:
			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "unsupported cloud type",
				Error:   "unsupported cloud type",
			}
		}

		profile, err := defaults.GetProfile(distribution, createClusterRequest.ProfileName)
		if err != nil {
			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "error during getting profile",
				Error:   err.Error(),
			}
		}

		logger.Info("create profile response")
		profileResponse := profile.GetProfile()

		logger.Info("create cluster request from profile")
		newRequest, err := profileResponse.CreateClusterRequest(createClusterRequest)
		if err != nil {
			logger.Errorf("error during getting cluster request from profile: %s", err.Error())

			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error creating request from profile",
				Error:   err.Error(),
			}
		}

		createClusterRequest = newRequest

		logger.Infof("modified clusterRequest: %v", createClusterRequest)
	}

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

	creationCtx := cluster.CreationContext{
		OrganizationID:  organizationID,
		UserID:          userID,
		Name:            createClusterRequest.Name,
		SecretID:        createClusterRequest.SecretId,
		SecretIDs:       createClusterRequest.SecretIds,
		Provider:        createClusterRequest.Cloud,
		PostHooks:       postHooks,
		ExternalBaseURL: a.externalBaseURL,
	}

	creator := cluster.NewClusterCreator(createClusterRequest, commonCluster, a.workflowClient)

	commonCluster, err = a.clusterManager.CreateCluster(ctx, creationCtx, creator)

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
