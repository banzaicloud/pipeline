package api

import (
	"fmt"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	"net/http"
	"strings"
)

// SecretNotFoundError signals that a given secret was not found
type SecretNotFoundError struct {
	errMessage string
}

// Error returns error message as string
func (err SecretNotFoundError) Error() string {
	return err.errMessage
}

// ListObjectStoreBuckets returns the list of object storage buckets (object storage container in case of Azure)
// that can be accessed with the credentials from the given secret
func ListObjectStoreBuckets(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "ListObjectStoreBuckets"})

	organization := auth.GetCurrentOrganization(c.Request)
	organizationId := organization.IDString()

	secretId := c.GetHeader("secretId")
	if len(secretId) == 0 {
		replyWithErrorResponse(c, requiredHeaderParamMissingErrorResponse("secretId"))
		return
	}

	log.Debugf("secretId=%s", secretId)

	cloudType := c.Query("cloudType")
	if len(cloudType) == 0 {
		replyWithErrorResponse(c, requiredQueryParamMissingErrorResponse("cloudType"))
		return
	}

	log.Infof("Retrieving object store buckets: organisation id=%s", organizationId)

	retrievedSecret, err := getValidatedSecret(organizationId, secretId, cloudType)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	objectStore, err := objectstore.NewObjectStore(cloudType, retrievedSecret, organization)
	if err != nil {
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	if cloudType == constants.Amazon {
		location := c.Query("location")

		if len(location) == 0 {
			replyWithErrorResponse(c, requiredQueryParamMissingErrorResponse("location"))
			return
		}

		if err = objectStore.WithRegion(location); err != nil {
			replyWithErrorResponse(c, errorResponseFrom(err))
			return
		}
	}

	bucketList, err := objectStore.ListBuckets()

	if err != nil {
		log.Errorf("Retrieving object store buckets: organisation id=%s failed: %s", organizationId, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	c.JSON(http.StatusOK, bucketList)
	return
}

// CreateObjectStoreBuckets creates an objectstore bucket (blob container in case of Azure)
// and also creating all requirements for them (eg.; ResourceGroup and StorageAccunt in case of Azure)
// these informations are also stored to a database
func CreateObjectStoreBuckets(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})
	log.Info("Creating bucket...")
	log.Info("Get organization id from params")
	organization := auth.GetCurrentOrganization(c.Request)
	organizationID := organization.IDString()
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
	cloudType, err := determineCloudProviderFromRequest(createBucketRequest)
	if err != nil {
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	log.Debug("Validating secret")
	retrievedSecret, err := getValidatedSecret(organizationID, createBucketRequest.SecretId,
		cloudType)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}
	log.Debug("Secret validation successful")
	log.Debug("Create CommonObjectStoreBuckets")
	objectStore, err :=
		objectstore.NewObjectStore(cloudType, retrievedSecret, organization)
	if err != nil {
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}
	log.Debug("CommonObjectStoreBuckets created")
	log.Debug("Bucket creation started")
	c.JSON(http.StatusAccepted, components.CreateBucketResponse{
		Name: createBucketRequest.Name,
	})
	if cloudType == constants.Amazon {
		objectStore.WithRegion(createBucketRequest.Properties.CreateAmazonObjectStoreBucketProperties.Location)
	}
	if cloudType == constants.Google {
		objectStore.WithRegion(createBucketRequest.Properties.CreateGoogleObjectStoreBucketProperties.Location)
	}
	if cloudType == constants.Azure {
		objectStore.WithRegion(createBucketRequest.Properties.CreateAzureObjectStoreBucketProperties.Location)
		objectStore.WithResourceGroup(createBucketRequest.Properties.CreateAzureObjectStoreBucketProperties.ResourceGroup)
		objectStore.WithStorageAccount(createBucketRequest.Properties.CreateAzureObjectStoreBucketProperties.StorageAccount)
	}

	go objectStore.CreateBucket(createBucketRequest.Name)
	return
}

// CheckObjectStoreBucket checks if the given there is a bucket exists with the given name
func CheckObjectStoreBucket(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "CheckObjectStoreBucket"})
	cloudType := c.Query("cloudType")
	bucketName := c.Param("name")
	log.Infof("Check if the bucket %s exists", bucketName)
	log.Info("Get organization id from params")
	organization := auth.GetCurrentOrganization(c.Request)
	organizationID := organization.IDString()
	log.Infof("Organization id: %s", organizationID)
	secretId := c.GetHeader("secretId")
	if len(secretId) == 0 {
		c.Status(requiredHeaderParamMissingErrorResponse("secretId").Code)
		return
	}
	log.Debugf("secretId=%s", secretId)

	retrievedSecret, err := getValidatedSecret(organizationID, secretId, cloudType)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		c.Status(errorResponseFrom(err).Code)
		return
	}
	log.Debug("Create CommonObjectStoreBuckets")
	objectStore, err :=
		objectstore.NewObjectStore(cloudType, retrievedSecret, organization)
	if err != nil {
		log.Errorf("Instantiating object store client for cloudType=%s failed: %s", cloudType, err.Error())
		c.Status(errorResponseFrom(err).Code)
		return
	}
	if cloudType == constants.Azure {
		resourceGroup := c.Query("resourceGroup")
		if len(resourceGroup) == 0 {
			c.Status(requiredQueryParamMissingErrorResponse("resourceGroup").Code)
			return
		}

		storageAccount := c.Query("storageAccount")
		if len(storageAccount) == 0 {
			c.Status(requiredQueryParamMissingErrorResponse("storageAccount").Code)
			return
		}

		if err = objectStore.WithResourceGroup(resourceGroup); err != nil {
			c.Status(errorResponseFrom(err).Code)
			return
		}

		if err = objectStore.WithStorageAccount(storageAccount); err != nil {
			c.Status(errorResponseFrom(err).Code)
			return
		}
	}
	log.Debug("CommonObjectStoreBuckets created")
	err = objectStore.CheckBucket(bucketName)
	if err != nil {
		c.Status(errorResponseFrom(err).Code)
		return
	}
	c.Status(http.StatusOK)
}

// DeleteObjectStoreBucket deletes object storage buckets (object storage container in case of Azure)
// that can be accessed with the credentials from the given secret
func DeleteObjectStoreBucket(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteObjectStoreBucket"})

	name := c.Param("name")

	organization := auth.GetCurrentOrganization(c.Request)
	organizationId := organization.IDString()
	secretId := c.GetHeader("secretId")
	if len(secretId) == 0 {
		replyWithErrorResponse(c, requiredHeaderParamMissingErrorResponse("secretId"))
		return
	}

	log.Debugf("secretId=%s", secretId)

	cloudType := c.Query("cloudType")
	if len(cloudType) == 0 {
		replyWithErrorResponse(c, requiredQueryParamMissingErrorResponse("cloudType"))
		return
	}

	retrievedSecret, err := getValidatedSecret(organizationId, secretId, cloudType)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	log.Infof("Deleting object store bucket: organisation id=%s, bucket=%s", organizationId, name)

	objectStore, err := objectstore.NewObjectStore(cloudType, retrievedSecret, organization)
	if err != nil {
		log.Errorf("Instantiating object store client for cloudType=%s failed: %s", cloudType, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	if cloudType == constants.Azure {
		resourceGroup := c.Query("resourceGroup")
		if len(resourceGroup) == 0 {
			replyWithErrorResponse(c, requiredQueryParamMissingErrorResponse("resourceGroup"))
			return
		}

		storageAccount := c.Query("storageAccount")
		if len(storageAccount) == 0 {
			replyWithErrorResponse(c, requiredQueryParamMissingErrorResponse("storageAccount"))
			return
		}

		if err = objectStore.WithResourceGroup(resourceGroup); err != nil {
			replyWithErrorResponse(c, errorResponseFrom(err))
			return
		}

		if err = objectStore.WithStorageAccount(storageAccount); err != nil {
			replyWithErrorResponse(c, errorResponseFrom(err))
			return
		}
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		log.Errorf("Deleting object store bucket: organisation id=%s, bucket=%s failed: %s", organizationId, name, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	log.Infof("Object store bucket: organisation id=%s, bucket=%s deleted", organizationId, name)

}

// errorResponseFrom translates the given error into a components.ErrorResponse
func errorResponseFrom(err error) *components.ErrorResponse {

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
	if err == constants.ErrorNotSupportedCloudType {
		return &components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	switch err.(type) {
	case SecretNotFoundError, secret.MissmatchError:
		return &components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	case objectstore.ManagedBucketNotFoundError:
		return &components.ErrorResponse{
			Code:    http.StatusNotFound,
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

// requiredQueryParamMissingErrorResponse creates an components.ErrorResponse denoting missing required query param
func requiredQueryParamMissingErrorResponse(queryParamName string) *components.ErrorResponse {
	return &components.ErrorResponse{
		Code:    http.StatusBadRequest,
		Error:   "Query parameter required.",
		Message: fmt.Sprintf("Required query parameter '%s' is missing", queryParamName),
	}
}

// requiredHeaderParamMissingErrorResponse creates an components.ErrorResponse denoting missing required header param
func requiredHeaderParamMissingErrorResponse(headerParamName string) *components.ErrorResponse {
	return &components.ErrorResponse{
		Code:    http.StatusBadRequest,
		Error:   "Header parameter required.",
		Message: fmt.Sprintf("Required header parameter '%s' is missing", headerParamName),
	}
}

// getValidatedSecret looks up the secret by secretId under the given organisation
// it also verifies if the found secret is of appropriate type for the given cloud provider
func getValidatedSecret(organizationId, secretId, cloudType string) (*secret.SecretsItemResponse, error) {

	// Validate Secret
	retrievedSecret, err := secret.Store.Get(organizationId, secretId)

	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			return nil, SecretNotFoundError{errMessage: err.Error()}
		}

		return nil, err
	}

	if err := utils.ValidateCloudType(retrievedSecret.Type); err != nil {
		return nil, err
	}

	if err := retrievedSecret.ValidateSecretType(cloudType); err != nil {
		return nil, err
	}

	return retrievedSecret, nil
}

func determineCloudProviderFromRequest(req components.CreateBucketRequest) (string, error) {
	if req.Properties.CreateAzureObjectStoreBucketProperties != nil {
		return constants.Azure, nil
	}
	if req.Properties.CreateAmazonObjectStoreBucketProperties != nil {
		return constants.Amazon, nil
	}
	if req.Properties.CreateGoogleObjectStoreBucketProperties != nil {
		return constants.Google, nil
	}
	return "", constants.ErrorNotSupportedCloudType
}
