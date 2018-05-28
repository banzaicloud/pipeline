package api

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
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
	retrievedSecret, err := secret.Store.Get(organizationID, secretID)
	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			c.JSON(http.StatusBadRequest, components.ErrorResponse{
				Code:    http.StatusBadRequest,
				Error:   err.Error(),
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})
		return
	}
	commonObjectStore, err := objectstore.ListCommonObjectStoreBuckets(retrievedSecret)
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
				Code:    http.StatusBadRequest,
				Error:   err.Error(),
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})
		return
	}
	log.Debug("Secret validation successful")
	log.Debug("Create CommonObjectStoreBuckets from request")
	commonObjectStore, err :=
		objectstore.CreateCommonObjectStoreBuckets(createBucketRequest, retrievedSecret)
	if err != nil {
		c.JSON(http.StatusNotImplemented, components.ErrorResponse{
			Code:    http.StatusNotImplemented,
			Error:   err.Error(),
			Message: err.Error(),
		})
	}
	log.Debug("CommonObjectStoreBuckets created")
	log.Debug("Bucket creation started")
	if err = commonObjectStore.CreateBucket(createBucketRequest.Name); err != nil {
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		})
		return
	}
	return
}

func deleteObjectStoreCommon(c *gin.Context, cloudType string) (string, *secret.SecretsItemResponse) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteBucket"})

	name := c.Param("name")
	log.Infof("Deleting bucket...%s", name)

	log.Info("Get organization id from params")
	organizationId := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationId)

	secretId := c.GetHeader("SecretId")
	// Validate Secret
	retrievedSecret, err := secret.Store.Get(organizationId, secretId)

	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			c.JSON(http.StatusBadRequest, components.ErrorResponse{
				Code:    http.StatusBadRequest,
				Error:   err.Error(),
				Message: err.Error(),
			})
			return "", nil
		}
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})
		return "", nil
	}

	if err := utils.ValidateCloudType(cloudType); err != nil {
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		})

		return "", nil
	}

	if err := retrievedSecret.ValidateSecretType(cloudType); err != nil {
		log.Infof("The passed in secret %s has wrong type %s ! Expected secret type %s",
			retrievedSecret.SecretType, retrievedSecret.SecretType, cloudType)

		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		})

		return "", nil
	}

	log.Debug("Secret validation successful")

	return name, retrievedSecret
}

// DeleteObjectStoreBucketGoogle deletes the GS bucket identified by name
func DeleteObjectStoreBucketGoogle(c *gin.Context) {

	name, retrievedSecret := deleteObjectStoreCommon(c, constants.Google)

	objectStore, err := objectstore.NewGoogleObjectStore(retrievedSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})

		return
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})
	}
}

// DeleteObjectStoreBucketAmazon deletes the S3 bucket identified by name
// from the given region
func DeleteObjectStoreBucketAmazon(c *gin.Context) {

	name, retrievedSecret := deleteObjectStoreCommon(c, constants.Amazon)
	region := c.GetHeader("region")

	objectStore, err := objectstore.NewAmazonObjectStore(retrievedSecret, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})

		return
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})
	}

}

// DeleteObjectStoreBucketAzure deletes the Azure Blob Container identified by name
// from the given resource group and storage account
func DeleteObjectStoreBucketAzure(c *gin.Context) {

	name, retrievedSecret := deleteObjectStoreCommon(c, constants.Azure)

	resourceGroup := c.GetHeader("ResourceGroup")
	storageAccount := c.GetHeader("StorageAccount")

	objectStore, err := objectstore.NewAzureObjectStore(retrievedSecret, resourceGroup, storageAccount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})

		return
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		})
	}
}
