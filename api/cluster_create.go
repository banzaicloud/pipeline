package api

import (
	"context"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//CreateClusterRequest gin handler
func CreateClusterRequest(c *gin.Context) {
	//TODO refactor logging here

	log.Info("Cluster creation started")

	log.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createClusterRequest pkgCluster.CreateClusterRequest
	if err := c.BindJSON(&createClusterRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	orgID := auth.GetCurrentOrganization(c.Request).ID
	userID := auth.GetCurrentUser(c.Request).ID

	ph := getPostHookFunctions(createClusterRequest.PostHooks)
	ctx := ginutils.Context(context.Background(), c)
	commonCluster, err := CreateCluster(ctx, &createClusterRequest, orgID, userID, ph)
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
func CreateCluster(
	ctx context.Context,
	createClusterRequest *pkgCluster.CreateClusterRequest,
	organizationID uint,
	userID uint,
	postHooks []cluster.PostFunctioner,
) (cluster.CommonCluster, *pkgCommon.ErrorResponse) {
	logger := log.WithFields(logrus.Fields{
		"organization": organizationID,
		"user":         userID,
		"cluster":      createClusterRequest.Name,
	})

	// TODO: refactor profile handling as well?
	if len(createClusterRequest.ProfileName) != 0 {
		logger = logger.WithField("profile", createClusterRequest.ProfileName)

		logger.Info("fill data from profile")

		profile, err := defaults.GetProfile(createClusterRequest.Cloud, createClusterRequest.ProfileName)
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

	logger.Info("Creating new entry with cloud type: ", createClusterRequest.Cloud)

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

	// TODO: move these to a struct and create them only once upon application init
	clusters := intCluster.NewClusters(config.DB())
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(clusters, secretValidator, log)

	creationCtx := cluster.CreationContext{
		OrganizationID: organizationID,
		UserID:         userID,
		Name:           createClusterRequest.Name,
		SecretID:       createClusterRequest.SecretId,
		Provider:       createClusterRequest.Cloud,
		PostHooks:      postHooks,
	}

	creator := cluster.NewCommonClusterCreator(createClusterRequest, commonCluster)

	commonCluster, err = clusterManager.CreateCluster(ctx, creationCtx, creator)

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
