package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"strings"
	"net/http"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/objectstore"
	"github.com/pkg/errors"
)



func ListObjectStoreBuckets(c *gin.Context) {
	//TODO Add proper logging
	log := logger.WithFields(logrus.Fields{"tag": "ListBuckets"})
	log.Info("Listing buckets...")
	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	secretID := c.Param("secretid")

	// Validate Secret
	secret, err := secret.Store.Get(organizationID, secretID)
	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			c.JSON(http.StatusBadRequest, components.ErrorResponse{
				Code: http.StatusBadRequest,
				Error: err.Error(),
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code: http.StatusInternalServerError,
			Error: err.Error(),
			Message: err.Error(),
		})
		return
	}
	commonObjectStore, err := objectstore.ListCommonObjectStoreBuckets(secret)
	if err != nil {

	}
	err = commonObjectStore.ListBuckets()
}

func CreateObjectStoreBuckets(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})
	log.Info("Creating bucket...")
	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	log.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createBucketRequest components.CreateBucketRequest
	if err := c.BindJSON(&createBucketRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	log.Debug("Validating secret")
	retrievedSecret, err := secret.Store.Get(organizationID, createBucketRequest.SecretId)
	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			c.JSON(http.StatusBadRequest, components.ErrorResponse{
				Code: http.StatusBadRequest,
				Error: err.Error(),
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code: http.StatusInternalServerError,
			Error: err.Error(),
			Message: err.Error(),
		})
		return
	}
	log.Debug("Secret validation successful")
	log.Debug("Create CommonObjectStoreBuckets from request")
	commonObjectStore, err :=
		objectstore.CreateCommonObjectStoreBuckets(createBucketRequest.Properties.Name ,createBucketRequest.Properties.Location, retrievedSecret)
	if err != nil {
		c.JSON(http.StatusNotImplemented, components.ErrorResponse{
			Code: http.StatusNotImplemented,
			Error: err.Error(),
			Message: err.Error(),
		})
	}
	log.Debug("CommonObjectStoreBuckets created")
	log.Debug("Bucket creation started")
	if err = commonObjectStore.CreateBucket(); err != nil {
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code: http.StatusBadRequest,
			Error: err.Error(),
			Message: err.Error(),
		})
		return
	}
	return
}
