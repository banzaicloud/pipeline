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
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/internal/providers"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgProviders "github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
)

const (
	// header key constants
	secretNameHeader = "secretName"
	secretIdHeader   = "secretId"
)

// ListAllBuckets handles bucket list requests. The handler method directs the flow to the appropriate retrieval
// strategy based on the request header details
func ListAllBuckets(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	if hasSecret(c) {
		// fallback to the initial implementation
		logger.Debug("proceeding to listing buckets based on the provided secret")
		ListBuckets(c)
		return
	}

	logger.Debug("proceeding to listing managed buckets")
	ListManagedBuckets(c)
	return

}

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
	case pkgProviders.Alibaba, pkgProviders.Amazon:
		location, ok := ginutils.RequiredQueryOrAbort(c, "location")
		if !ok {
			logger.Debug("missing location")

			return
		}

		objectStoreCtx.Location = location
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		errorHandler.Handle(err)
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

// ListManagedBuckets lists managed buckets for the user when no secret is provided
func ListManagedBuckets(c *gin.Context) {

	type secretData struct {
		SecretId   string `json:"secretId"`
		SecretName string `json:"secretName"`
	}
	type bucketWithSecretName struct {
		*objectstore.BucketInfo
		SecretRef  string      `json:"secretId,-,omitempty"`
		SecretInfo *secretData `json:"secretInfo"`
	}

	logger := correlationid.Logger(log, c)
	organization := auth.GetCurrentOrganization(c.Request)

	allProviders := []string{
		pkgProviders.Alibaba,
		pkgProviders.Amazon,
		pkgProviders.Azure,
		pkgProviders.Google,
		pkgProviders.Oracle,
	}

	secretNamesNeeded := c.Query("fields") == "name"

	allBuckets := make([]*objectstore.BucketInfo, 0)
	for _, cloudType := range allProviders {
		logger.Debugf("retrieving buckets for provider: %s", cloudType)

		objectStoreCtx := &providers.ObjectStoreContext{
			Provider:     cloudType,
			Organization: organization,
		}

		objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
		if err != nil {
			logger.Warnf("error creating object store, managed buckets not retrieved for cloud type: %s", cloudType)
			errorHandler.Handle(err)
			continue
		}

		bucketList, err := objectStore.ListManagedBuckets()
		if err != nil {
			logger.Errorf("retrieving object store buckets failed: %s", err.Error())
			continue
		}

		allBuckets = append(allBuckets, bucketList...)

	}

	if secretNamesNeeded {
		logger.Debug("decorating buckets with secret names")
		withSecretName := make([]*bucketWithSecretName, 0)
		for _, bucket := range allBuckets {

			secret, err := secret.Store.Get(organization.ID, bucket.SecretRef)
			if err != nil {
				errorHandler.Handle(err)
			}

			withSecretName = append(withSecretName, &bucketWithSecretName{
				BucketInfo: bucket,
				SecretInfo: &secretData{SecretName: secret.Name, SecretId: secret.ID}})
		}

		c.JSON(http.StatusOK, withSecretName)
		return
	}

	c.JSON(http.StatusOK, allBuckets)

}

// CreateBucket creates an objectstore bucket (blob container in case of Azure)
// and also creates all requirements for them (eg.; ResourceGroup and StorageAccount in case of Azure).
// These information are also stored to a database.
func CreateBucket(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	organization := auth.GetCurrentOrganization(c.Request)

	logger = logger.WithField("organization", organization.ID)

	logger.Debug("bind json into CreateClusterRequest struct")

	var createBucketRequest CreateBucketRequest
	if err := c.BindJSON(&createBucketRequest); err != nil {
		logger.Error(errors.Wrap(err, "Error parsing request"))

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})

		return
	}

	if createBucketRequest.SecretId == "" {
		if createBucketRequest.SecretName == "" {
			c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "either secretId or secretName has to be set",
			})
			return
		}

		createBucketRequest.SecretId = secret.GenerateSecretIDFromName(createBucketRequest.SecretName)
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
	case pkgProviders.Alibaba:
		objectStoreCtx.Location = createBucketRequest.Properties.Alibaba.Location

	case pkgProviders.Amazon:
		objectStoreCtx.Location = createBucketRequest.Properties.Amazon.Location

	case pkgProviders.Google:
		objectStoreCtx.Location = createBucketRequest.Properties.Google.Location

	case pkgProviders.Azure:
		objectStoreCtx.Location = createBucketRequest.Properties.Azure.Location
		objectStoreCtx.ResourceGroup = createBucketRequest.Properties.Azure.ResourceGroup
		objectStoreCtx.StorageAccount = createBucketRequest.Properties.Azure.StorageAccount

	case pkgProviders.Oracle:
		objectStoreCtx.Location = createBucketRequest.Properties.Oracle.Location
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	logger.Debug("bucket creation started")

	c.JSON(http.StatusAccepted, CreateBucketResponse{
		BucketName: createBucketRequest.Name,
	})

	go func() {
		defer emperror.HandleRecover(errorHandler)

		err := objectStore.CreateBucket(createBucketRequest.Name)
		if err != nil {
			errorHandler.Handle(err)
		}
	}()

	return
}

// CheckBucket checks if the given there is a bucket exists with the given name
func CheckBucket(c *gin.Context) {
	logger := correlationid.Logger(log, c)

	bucketName := c.Param("name")
	logger = logger.WithField("bucket", bucketName)

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
	case pkgProviders.Alibaba, pkgProviders.Amazon, pkgProviders.Oracle:
		location, ok := ginutils.RequiredQueryOrAbort(c, "location")
		if !ok {
			logger.Debug("missing location")

			return
		}

		objectStoreCtx.Location = location

	case pkgProviders.Azure:
		resourceGroup, ok := ginutils.RequiredQueryOrAbort(c, "resourceGroup")
		if !ok {
			logger.Debug("missing resource group")

			return
		}

		storageAccount, ok := ginutils.RequiredQueryOrAbort(c, "storageAccount")
		if !ok {
			logger.Debug("missing storage account")

			return
		}

		objectStoreCtx.ResourceGroup = resourceGroup
		objectStoreCtx.StorageAccount = storageAccount
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		errorHandler.Handle(err)
		c.Status(errorResponseFrom(err).Code)

		return
	}

	err = objectStore.CheckBucket(bucketName)
	if err != nil {
		errorHandler.Handle(err)
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
	logger = logger.WithField("bucket", bucketName)

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
	case pkgProviders.Oracle:
		location, ok := ginutils.RequiredQueryOrAbort(c, "location")
		if !ok {
			logger.Debug("missing location")

			return
		}

		objectStoreCtx.Location = location

	case pkgProviders.Azure:
		resourceGroup, ok := ginutils.RequiredQueryOrAbort(c, "resourceGroup")
		if !ok {
			logger.Debug("missing resource group")

			return
		}

		storageAccount, ok := ginutils.RequiredQueryOrAbort(c, "storageAccount")
		if !ok {
			logger.Debug("missing storage account")

			return
		}

		objectStoreCtx.ResourceGroup = resourceGroup
		objectStoreCtx.StorageAccount = storageAccount
	}

	objectStore, err := providers.NewObjectStore(objectStoreCtx, logger)
	if err != nil {
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	if err = objectStore.DeleteBucket(bucketName); err != nil {
		errorHandler.Handle(err)
		ginutils.ReplyWithErrorResponse(c, errorResponseFrom(err))

		return
	}

	logger.Infof("object store bucket deleted")
}

// hasSecret checks the header for secret references, returns true in case one of the following headers are found:
// - secretName
// - secretId
// otherwise returns false
func hasSecret(c *gin.Context) bool {
	return c.GetHeader(secretNameHeader) != "" || c.GetHeader(secretIdHeader) != ""
}
func getBucketContext(c *gin.Context, logger logrus.FieldLogger) (*auth.Organization, *secret.SecretItemResponse, string, bool) {
	organization := auth.GetCurrentOrganization(c.Request)

	var secretID string
	var ok bool

	secretName := c.GetHeader(secretNameHeader)
	if secretName != "" {
		secretID = secret.GenerateSecretIDFromName(secretName)
	} else {
		secretID, ok = ginutils.GetRequiredHeader(c, secretIdHeader)
		if !ok {
			return nil, nil, "", false
		}
	}

	provider, ok := ginutils.RequiredQueryOrAbort(c, "cloudType")
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

	if err := pkgProviders.ValidateProvider(retrievedSecret.Type); err != nil {
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
func errorResponseFrom(err error) *pkgCommon.ErrorResponse {
	if objectstore.IsNotFoundError(err) {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	// google specific errors
	if googleApiErr, ok := err.(*googleapi.Error); ok {
		return &pkgCommon.ErrorResponse{
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

		return &pkgCommon.ErrorResponse{
			Code:    code,
			Error:   awsErr.Error(),
			Message: awsErr.Message(),
		}
	}

	// azure specific errors
	if azureErr, ok := err.(validation.Error); ok {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	if azureErr, ok := err.(azblob.StorageError); ok {
		serviceCode := fmt.Sprint(azureErr.ServiceCode())

		return &pkgCommon.ErrorResponse{
			Code:    azureErr.Response().StatusCode,
			Error:   azureErr.Error(),
			Message: serviceCode,
		}
	}

	if azureErr, ok := err.(autorest.DetailedError); ok {
		if azureErr.Original != nil {
			if azureOrigErr, ok := azureErr.Original.(*azure.RequestError); ok {
				return &pkgCommon.ErrorResponse{
					Code:    azureErr.Response.StatusCode,
					Error:   azureOrigErr.ServiceError.Error(),
					Message: azureOrigErr.ServiceError.Message,
				}
			}

			return &pkgCommon.ErrorResponse{
				Code:    azureErr.Response.StatusCode,
				Error:   azureErr.Original.Error(),
				Message: azureErr.Message,
			}
		}

		return &pkgCommon.ErrorResponse{
			Code:    azureErr.Response.StatusCode,
			Error:   azureErr.Error(),
			Message: azureErr.Message,
		}
	}

	// pipeline specific errors
	if err == pkgErrors.ErrorNotSupportedCloudType {
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}

	switch err.(type) {
	case SecretNotFoundError, secret.MissmatchError:
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   err.Error(),
			Message: err.Error(),
		}
	default:
		return &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Error:   err.Error(),
			Message: err.Error(),
		}
	}
}
