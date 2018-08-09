package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/objectstore"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/storage"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"google.golang.org/api/googleapi"
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

	organization := auth.GetCurrentOrganization(c.Request)
	organizationID := organization.ID

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

	log.Infof("Retrieving object store buckets: organization id=%d", organizationID)

	retrievedSecret, err := getValidatedSecret(organizationID, secretId, cloudType)
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

	if cloudType == pkgCluster.Amazon {
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
		log.Errorf("Retrieving object store buckets: organization id=%d failed: %s", organizationID, err.Error())
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
	log.Info("Creating bucket...")
	log.Info("Get organization id from params")
	organization := auth.GetCurrentOrganization(c.Request)
	organizationID := organization.ID
	log.Infof("Organization id: %d", organizationID)

	log.Debug("Bind json into CreateClusterRequest struct")
	// bind request body to struct
	var createBucketRequest storage.CreateBucketRequest
	if err := c.BindJSON(&createBucketRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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
	c.JSON(http.StatusAccepted, storage.CreateBucketResponse{
		Name: createBucketRequest.Name,
	})
	if cloudType == pkgCluster.Amazon {
		objectStore.WithRegion(createBucketRequest.Properties.CreateAmazonObjectStoreBucketProperties.Location)
	}
	if cloudType == pkgCluster.Google {
		objectStore.WithRegion(createBucketRequest.Properties.CreateGoogleObjectStoreBucketProperties.Location)
	}
	if cloudType == pkgCluster.Azure {
		objectStore.WithRegion(createBucketRequest.Properties.Azure.Location)
		objectStore.WithResourceGroup(createBucketRequest.Properties.Azure.ResourceGroup)
		objectStore.WithStorageAccount(createBucketRequest.Properties.Azure.StorageAccount)
	}
	if cloudType == pkgCluster.Oracle {
		objectStore.WithRegion(createBucketRequest.Properties.CreateOracleObjectStoreBucketProperties.Location)
	}

	go objectStore.CreateBucket(createBucketRequest.Name)
	return
}

// CheckObjectStoreBucket checks if the given there is a bucket exists with the given name
func CheckObjectStoreBucket(c *gin.Context) {
	cloudType := c.Query("cloudType")
	bucketName := c.Param("name")
	log.Infof("Check if the bucket %s exists", bucketName)
	log.Info("Get organization id from params")
	organization := auth.GetCurrentOrganization(c.Request)
	organizationID := organization.ID
	log.Infof("Organization id: %d", organizationID)
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
	if cloudType == pkgCluster.Azure {
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
	if cloudType == pkgCluster.Oracle {
		location := c.Query("location")
		if len(location) == 0 {
			c.Status(requiredQueryParamMissingErrorResponse("location").Code)
			return
		}
		if err = objectStore.WithRegion(location); err != nil {
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

	name := c.Param("name")

	organization := auth.GetCurrentOrganization(c.Request)
	organizationID := organization.ID
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

	retrievedSecret, err := getValidatedSecret(organizationID, secretId, cloudType)
	if err != nil {
		log.Errorf("Secret validation failed: %s", err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	log.Infof("Deleting object store bucket: organization id=%d, bucket=%s", organizationID, name)

	objectStore, err := objectstore.NewObjectStore(cloudType, retrievedSecret, organization)
	if err != nil {
		log.Errorf("Instantiating object store client for cloudType=%s failed: %s", cloudType, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	if cloudType == pkgCluster.Azure {
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
	if cloudType == pkgCluster.Oracle {
		location := c.Query("location")
		if len(location) == 0 {
			replyWithErrorResponse(c, requiredQueryParamMissingErrorResponse("location"))
			return
		}
		if err = objectStore.WithRegion(location); err != nil {
			c.Status(errorResponseFrom(err).Code)
			return
		}
	}

	if err = objectStore.DeleteBucket(name); err != nil {
		log.Errorf("Deleting object store bucket: organization id=%d, bucket=%s failed: %s", organizationID, name, err.Error())
		replyWithErrorResponse(c, errorResponseFrom(err))
		return
	}

	log.Infof("Object store bucket: organization id=%d, bucket=%s deleted", organizationID, name)

}

// errorResponseFrom translates the given error into a components.ErrorResponse
func errorResponseFrom(err error) *common.ErrorResponse {

	// google specific errors
	if googleApiErr, ok := err.(*googleapi.Error); ok {
		return &common.ErrorResponse{
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

		return &common.ErrorResponse{
			Code:    code,
			Error:   awsErr.Error(),
			Message: awsErr.Message(),
		}
	}

	// azure specific errors
	if azureErr, ok := err.(validation.Error); ok {
		return &common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	if azureErr, ok := err.(azblob.StorageError); ok {
		serviceCode := fmt.Sprint(azureErr.ServiceCode())

		return &common.ErrorResponse{
			Code:    azureErr.Response().StatusCode,
			Error:   azureErr.Error(),
			Message: serviceCode,
		}
	}

	if azureErr, ok := err.(autorest.DetailedError); ok {
		if azureErr.Original != nil {
			if azureOrigErr, ok := azureErr.Original.(*azure.RequestError); ok {
				return &common.ErrorResponse{
					Code:    azureErr.Response.StatusCode,
					Error:   azureOrigErr.ServiceError.Error(),
					Message: azureOrigErr.ServiceError.Message,
				}
			}

			return &common.ErrorResponse{
				Code:    azureErr.Response.StatusCode,
				Error:   azureErr.Original.Error(),
				Message: azureErr.Message,
			}
		}

		return &common.ErrorResponse{
			Code:    azureErr.Response.StatusCode,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	// pipeline specific errors
	if err == pkgErrors.ErrorNotSupportedCloudType {
		return &common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	if isNotFound(err) {
		return &common.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	switch err.(type) {
	case SecretNotFoundError, secret.MissmatchError:
		return &common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	case objectstore.ManagedBucketNotFoundError:
		return &common.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   err.Error(),
			Message: err.Error(),
		}
	default:
		return &common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}
}

func replyWithErrorResponse(c *gin.Context, errorResponse *common.ErrorResponse) {
	c.JSON(errorResponse.Code, errorResponse)
}

// requiredQueryParamMissingErrorResponse creates an components.ErrorResponse denoting missing required query param
func requiredQueryParamMissingErrorResponse(queryParamName string) *common.ErrorResponse {
	return &common.ErrorResponse{
		Code:    http.StatusBadRequest,
		Error:   "Query parameter required.",
		Message: fmt.Sprintf("Required query parameter '%s' is missing", queryParamName),
	}
}

// requiredHeaderParamMissingErrorResponse creates an components.ErrorResponse denoting missing required header param
func requiredHeaderParamMissingErrorResponse(headerParamName string) *common.ErrorResponse {
	return &common.ErrorResponse{
		Code:    http.StatusBadRequest,
		Error:   "Header parameter required.",
		Message: fmt.Sprintf("Required header parameter '%s' is missing", headerParamName),
	}
}

// getValidatedSecret looks up the secret by secretId under the given organisation
// it also verifies if the found secret is of appropriate type for the given cloud provider
func getValidatedSecret(organizationId uint, secretId, cloudType string) (*secret.SecretItemResponse, error) {

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

func determineCloudProviderFromRequest(req storage.CreateBucketRequest) (string, error) {
	if req.Properties.Azure != nil {
		return pkgCluster.Azure, nil
	}
	if req.Properties.CreateAmazonObjectStoreBucketProperties != nil {
		return pkgCluster.Amazon, nil
	}
	if req.Properties.CreateGoogleObjectStoreBucketProperties != nil {
		return pkgCluster.Google, nil
	}
	if req.Properties.CreateOracleObjectStoreBucketProperties != nil {
		return pkgCluster.Oracle, nil
	}
	return "", pkgErrors.ErrorNotSupportedCloudType
}
