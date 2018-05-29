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
	"google.golang.org/api/googleapi"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"fmt"
)


type  SecretNotFoundError struct {
	errMessage string
}

func (err SecretNotFoundError) Error() string {
	return err.errMessage
}


func ListObjectStoreBuckets(c *gin.Context) {
	//TODO Add proper logging
	log := logger.WithFields(logrus.Fields{"tag": "ListBuckets"})
	log.Info("Listing buckets...")
	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	secretId := c.Param("secretId")

	// Validate Secret
	retrievedSecret, err := secret.Store.Get(organizationID, secretId)
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

// getValidatedSecret looks up the secret by secretId under the given organisation
// it also verifies if the found secret is of appropriate type for the given cloud provider
func getValidatedSecret(organizationId, secretId, cloudType string) (*secret.SecretsItemResponse, error) {

	// Validate Secret
	retrievedSecret, err := secret.Store.Get(organizationId, secretId)

	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			return nil, SecretNotFoundError{ errMessage: err.Error() }
		}

		return nil, err
	}

	if err := utils.ValidateCloudType(cloudType); err != nil {
		return nil, err
	}

	if err := retrievedSecret.ValidateSecretType(cloudType); err != nil {
		return nil, err
	}

	return retrievedSecret, nil
}

// DeleteGoogleObjectStoteBucket deletes the GS bucket identified by name
func DeleteGoogleObjectStoteBucket(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteGoogleObjectStoteBucket"})

	name := c.Param("name")

	organizationId := auth.GetCurrentOrganization(c.Request).IDString()
	secretId := c.GetHeader("secretId")
	log.Debugf("secretId=%s", secretId)

	log.Infof("Deleting GS bucket: organisation id=%s, bucket=%s", organizationId, name)

	retrievedSecret, err := getValidatedSecret(organizationId, secretId, constants.Google)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	objectStore, err := objectstore.NewGoogleObjectStore(retrievedSecret)
	if err != nil {
		log.Errorf("Instantiating GoogleObjectStore failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		log.Errorf("Deleting GS bucket: organisation id=%s, bucket=%s failed: %s", organizationId, name, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	log.Infof("GS bucket: organisation id=%s, bucket=%s deleted", organizationId, name)
}

// DeleteAmazonObjectStoreBucket deletes the S3 bucket identified by name
// from the given region
func DeleteAmazonObjectStoreBucket(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteAmazonObjectStoreBucket"})

	name := c.Param("name")
	region := c.Param("region")


	organizationId := auth.GetCurrentOrganization(c.Request).IDString()
	secretId := c.GetHeader("secretId")
	log.Debugf("secretId=%s", secretId)

	log.Infof("Deleting S3 bucket: organisation id=%s, region=%s, bucket=%s", organizationId, region, name)

	retrievedSecret, err := getValidatedSecret(organizationId, secretId, constants.Amazon)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}


	objectStore, err := objectstore.NewAmazonObjectStore(retrievedSecret, region)
	if err != nil {
		log.Errorf("Instantiating AmazonObjectStore failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		log.Errorf("Deleting S3 bucket: organisation id=%s, region=%s, bucket=%s failed: %s", organizationId, region, name, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}
	log.Infof("S3 bucket: organisation id=%s, region=%s, bucket=%s deleted", organizationId, region, name)

}

// DeleteAzureObjectStoreContainer deletes the Azure Storage Container identified by name
// from the given resource group and storage account
func DeleteAzureObjectStoreContainer(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteAzureObjectStoreContainer"})

	name := c.Param("name")
	resourceGroup := c.Param("resourceGroup")
	storageAccount := c.Param("storageAccount")

	organizationId := auth.GetCurrentOrganization(c.Request).IDString()
	secretId := c.GetHeader("secretId")

	log.Infof("Deleting S3 bucket: organisation id=%s, resourceGroup=%s, storageAccount=%s, bucket=%s", organizationId, resourceGroup, storageAccount, name)

	retrievedSecret, err := getValidatedSecret(organizationId, secretId, constants.Azure)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	objectStore, err := objectstore.NewAzureObjectStore(retrievedSecret, resourceGroup, storageAccount)
	if err != nil {
		log.Errorf("Instantiating AzureObjectStore failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		log.Infof("Deleting S3 bucket: organisation id=%s, resourceGroup=%s, storageAccount=%s, bucket=%s", organizationId, resourceGroup, storageAccount, name)
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}
}


func errorResponseFrom(err error) *components.ErrorResponse {

	if err == constants.ErrorNotSupportedCloudType {
		return &components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	// google specific errors
	if googleApiErr, ok := err.(*googleapi.Error); ok {
		return &components.ErrorResponse{
			Code:    googleApiErr.Code,
			Error:   googleApiErr.Error(),
			Message: googleApiErr.Message,
		}
	}

	// aws specific errors
	if awsErr, ok := err.(awserr.Error); ok {
		code := http.StatusBadRequest
		if awsReqFailure, ok := err.(awserr.RequestFailure); ok {
			code = awsReqFailure.StatusCode()
		}

		return &components.ErrorResponse{
			Code:    code,
			Error:   awsErr.Error(),
			Message: awsErr.Message(),
		}
	}

	// azure specific errors
	if azureErr, ok := err.(validation.Error); ok {
		return &components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	if azureErr, ok := err.(azblob.StorageError); ok {
		serviceCode := fmt.Sprint(azureErr.ServiceCode())

		return &components.ErrorResponse{
			Code:    azureErr.Response().StatusCode,
			Error:   azureErr.Error(),
			Message: serviceCode,
		}
	}

	if azureErr, ok := err.(autorest.DetailedError); ok {
		if azureErr.Original != nil {
			if azureOrigErr, ok := azureErr.Original.(*azure.RequestError); ok {
				return &components.ErrorResponse{
					Code:    azureErr.Response.StatusCode,
					Error:   azureOrigErr.ServiceError.Error(),
					Message: azureOrigErr.ServiceError.Message,
				}
			}

			return &components.ErrorResponse{
				Code:    azureErr.Response.StatusCode,
				Error:   azureErr.Original.Error(),
				Message: azureErr.Message,
			}
		}

		return &components.ErrorResponse{
			Code:    azureErr.Response.StatusCode,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	// pipeline specific errors
	switch err.(type) {
	case SecretNotFoundError, secret.MissmatchError:
		return &components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	default:
		return &components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}
}

func replyWithErrorResponse(c *gin.Context, errorResponse *components.ErrorResponse) {
	c.JSON(errorResponse.Code, errorResponse)
}
