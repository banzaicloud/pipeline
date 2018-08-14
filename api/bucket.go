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
	"github.com/banzaicloud/pipeline/internal/gin/correlationid"
	"github.com/banzaicloud/pipeline/internal/gin/utils"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
)

// ListBuckets returns the list of object storage buckets (object storage container in case of Azure)
// that can be accessed with the credentials from the given secret.
func ListBuckets(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	organization, secret, cloudType, ok := getBucketContext(c, logger)
	if !ok {
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"secret":       secret.ID,
		"provider":     cloudType,
	})

	logger.Infof("retrieving object store buckets")

	objectStoreCtx := &providers.ObjectStoreContext{
		Provider:     cloudType,
		Secret:       secret,
		Organization: organization,
	}

	switch cloudType {
	case providers.Alibaba, providers.Amazon:
		location, ok := ginutils.RequiredQuery(c, "location")
		if !ok {
			logger.Debug("missing location")

			return
		}

		objectStoreCtx.Location = location
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	bucketList, err := objectStore.ListBuckets()

	if err != nil {
		logger.Errorf("retrieving object store buckets failed: %s", err.Error())
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	c.JSON(http.StatusOK, bucketList)
}

// CreateBucket creates an objectstore bucket (blob container in case of Azure)
// and also creates all requirements for them (eg.; ResourceGroup and StorageAccunt in case of Azure).
// These information are also stored to a database.
func CreateBucket(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	organization := auth.GetCurrentOrganization(c.Request)

	logger = logger.WithField("organization", organization.ID)

	logger.Debug("bind json into CreateClusterRequest struct")

	var createBucketRequest CreateBucketRequest
	if err := c.BindJSON(&createBucketRequest); err != nil {
		logger.Error(errors.Wrap(err, "Error parsing request"))

		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})

		return
	}

	cloudType, err := determineCloudProviderFromRequest(createBucketRequest)
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	logger = logger.WithFields(logrus.Fields{
		"secret":   createBucketRequest.SecretId,
		"provider": cloudType,
		"bucket":   createBucketRequest.Name,
	})

	logger.Debug("validating secret")
	retrievedSecret, err := getValidatedSecret(organization.ID, createBucketRequest.SecretId, cloudType)
	if err != nil {
		logger.Errorf("secret validation failed: %s", err.Error())
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	logger.Debug("secret validation successful")

	objectStoreCtx := &providers.ObjectStoreContext{
		Provider:     cloudType,
		Secret:       retrievedSecret,
		Organization: organization,
	}

	switch cloudType {
	case providers.Alibaba:
		objectStoreCtx.Location = createBucketRequest.Properties.Alibaba.Location

	case providers.Amazon:
		objectStoreCtx.Location = createBucketRequest.Properties.Amazon.Location

	case providers.Google:
		objectStoreCtx.Location = createBucketRequest.Properties.Google.Location

	case providers.Azure:
		objectStoreCtx.Location = createBucketRequest.Properties.Azure.Location
		objectStoreCtx.ResourceGroup = createBucketRequest.Properties.Azure.ResourceGroup
		objectStoreCtx.StorageAccount = createBucketRequest.Properties.Azure.StorageAccount

	case providers.Oracle:
		objectStoreCtx.Location = createBucketRequest.Properties.Oracle.Location
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	logger.Debug("bucket creation started")

	c.JSON(http.StatusAccepted, CreateBucketResponse{
		Name: createBucketRequest.Name,
	})

	go objectStore.CreateBucket(createBucketRequest.Name)

	return
}

// CheckBucket checks if the given there is a bucket exists with the given name
func CheckBucket(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	bucketName := c.Param("name")
	logger = logrus.WithField("bucket", bucketName)

	organization, secret, cloudType, ok := getBucketContext(c, logger)
	if !ok {
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"secret":       secret.ID,
		"provider":     cloudType,
	})

	objectStoreCtx := &providers.ObjectStoreContext{
		Provider:     cloudType,
		Secret:       secret,
		Organization: organization,
	}

	switch cloudType {
	case providers.Alibaba, providers.Amazon, providers.Oracle:
		location, ok := ginutils.RequiredQuery(c, "location")
		if !ok {
			logger.Debug("missing location")

			return
		}

		objectStoreCtx.Location = location

	case providers.Azure:
		resourceGroup, ok := ginutils.RequiredQuery(c, "resourceGroup")
		if !ok {
			logger.Debug("missing resource group")

			return
		}

		storageAccount, ok := ginutils.RequiredQuery(c, "storageAccount")
		if !ok {
			logger.Debug("missing storage account")

			return
		}

		objectStoreCtx.ResourceGroup = resourceGroup
		objectStoreCtx.StorageAccount = storageAccount
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		logger.Errorf("instantiating object store client for failed: %s", err.Error())
		c.Status(errorResponseFrom(err).Code)

		return
	}

	err = objectStore.CheckBucket(bucketName)
	if err != nil {
		c.Status(errorResponseFrom(err).Code)

		return
	}

	c.Status(http.StatusOK)
}

// DeleteBucket deletes object storage buckets (object storage container in case of Azure)
// that can be accessed with the credentials from the given secret
func DeleteBucket(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	bucketName := c.Param("name")
	logger = logrus.WithField("bucket", bucketName)

	organization, secret, cloudType, ok := getBucketContext(c, logger)
	if !ok {
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"secret":       secret.ID,
		"provider":     cloudType,
	})

	logger.Infof("deleting object store bucket")

	objectStoreCtx := &providers.ObjectStoreContext{
		Provider:     cloudType,
		Secret:       secret,
		Organization: organization,
	}

	switch cloudType {
	case providers.Oracle:
		location, ok := ginutils.RequiredQuery(c, "location")
		if !ok {
			logger.Debug("missing location")

			return
		}

		objectStoreCtx.Location = location

	case providers.Azure:
		resourceGroup, ok := ginutils.RequiredQuery(c, "resourceGroup")
		if !ok {
			logger.Debug("missing resource group")

			return
		}

		storageAccount, ok := ginutils.RequiredQuery(c, "storageAccount")
		if !ok {
			logger.Debug("missing storage account")

			return
		}

		objectStoreCtx.ResourceGroup = resourceGroup
		objectStoreCtx.StorageAccount = storageAccount
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		logger.Errorf("instantiating object store client failed: %s", err.Error())
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	if err = objectStore.DeleteBucket(bucketName); err != nil {
		logger.Errorf("deleting object store bucket failed: %s", err.Error())
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	logger.Infof("object store bucket deleted")
}

func getBucketContext(c *gin.Context, logger logrus.FieldLogger) (*auth.Organization, *secret.SecretItemResponse, string, bool) {
	organization := auth.GetCurrentOrganization(c.Request)

	secretID, ok := ginutils.GetRequiredHeader(c, "secretId")
	if !ok {
		logger.Debug("missing secret id")

		return nil, nil, "", false
	}

	provider, ok := ginutils.RequiredQuery(c, "cloudType")
	if !ok {
		logger.Debug("missing provider")

		return nil, nil, "", false
	}

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"secret":       secretID,
		"provider":     provider,
	})

	s, err := getValidatedSecret(organization.ID, secretID, provider)
	if err != nil {
		logger.Errorf("secret validation failed: %s", err.Error())
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return nil, nil, "", false
	}

	return organization, s, provider, true
}

// SecretNotFoundError signals that a given secret was not found
type SecretNotFoundError struct {
	errMessage string
}

// Error returns error message as string
func (err SecretNotFoundError) Error() string {
	return err.errMessage
}

// getValidatedSecret looks up the secret by secretId under the given organisation
// it also verifies if the found secret is of appropriate type for the given cloud provider
func getValidatedSecret(organizationId uint, secretId, cloudType string) (*secret.SecretItemResponse, error) {
	retrievedSecret, err := secret.Store.Get(organizationId, secretId)

	if err != nil {
		if strings.Contains(err.Error(), "there's no secret with this id") {
			return nil, SecretNotFoundError{errMessage: err.Error()}
		}

		return nil, err
	}

	if err := pkgCluster.ValidateCloudType(retrievedSecret.Type); err != nil {
		return nil, err
	}

	if err := retrievedSecret.ValidateSecretType(cloudType); err != nil {
		return nil, err
	}

	return retrievedSecret, nil
}

func determineCloudProviderFromRequest(req CreateBucketRequest) (string, error) {
	if req.Properties.Alibaba != nil {
		return pkgCluster.Alibaba, nil
	}
	if req.Properties.Azure != nil {
		return pkgCluster.Azure, nil
	}
	if req.Properties.Amazon != nil {
		return pkgCluster.Amazon, nil
	}
	if req.Properties.Google != nil {
		return pkgCluster.Google, nil
	}
	if req.Properties.Oracle != nil {
		return pkgCluster.Oracle, nil
	}
	return "", pkgErrors.ErrorNotSupportedCloudType
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
	default:
		return &common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}
}
