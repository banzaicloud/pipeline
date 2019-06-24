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
	"strconv"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ErrNotSupportedSecretType describe an error if the secret type is not supported
var ErrNotSupportedSecretType = errors.New("Not supported secret type")

// ValidateSecret validates the given secret
func ValidateSecret(c *gin.Context) {

	log.Info("start validation secret")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id [%d]", organizationID)

	secretID := getSecretID(c)
	log.Infof("secret id [%d]", secretID)

	secretItem, err := secret.RestrictedStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("Error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting secret",
			Error:   err.Error(),
		})
		return
	}

	version := int(secretItem.Version)

	if ok, _ := validateSecret(c, &secret.CreateSecretRequest{
		Name:      secretItem.Name,
		Type:      secretItem.Type,
		Values:    secretItem.Values,
		Tags:      secretItem.Tags,
		Version:   &version,
		UpdatedBy: secretItem.UpdatedBy,
	}, true, false); ok {
		c.Status(http.StatusOK)
	}

}

func validateSecret(c *gin.Context, createSecretRequest *secret.CreateSecretRequest, validate bool, new bool) (ok bool, validationError error) {

	ok = true
	log.Info("Start validation")
	verifier := verify.NewVerifier(createSecretRequest.Type, createSecretRequest.Values)

	if new {
		validationError = createSecretRequest.ValidateAsNew(verifier)
	} else {
		validationError = createSecretRequest.Validate(verifier)
	}

	if validationError != nil && validate {
		ok = false
		log.Errorf("Validation error: %s", validationError.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Validation error",
			Error:   validationError.Error(),
		})
	} else {
		log.Info("Validation passed")
	}

	return
}

// AddSecrets saves the given secret to vault
func AddSecrets(c *gin.Context) {

	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id: %d", organizationID)

	validateParam := c.DefaultQuery("validate", "true")
	validate, err := strconv.ParseBool(validateParam)
	if err != nil {
		validate = true
	}

	log.Infof("validate value %t", validate)

	var createSecretRequest secret.CreateSecretRequest
	if err := c.ShouldBind(&createSecretRequest); err != nil {
		log.Errorf("Error during binding CreateSecretRequest: %s", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during binding",
			Error:   err.Error(),
		})
		return
	}

	createSecretRequest.UpdatedBy = auth.GetCurrentUser(c.Request).Login

	//Check if the received value is base64 encoded if not encode it.
	if createSecretRequest.Values[secretTypes.K8SConfig] != "" {
		createSecretRequest.Values[secretTypes.K8SConfig] = utils.EncodeStringToBase64(createSecretRequest.Values[secretTypes.K8SConfig])
	}

	log.Info("Binding request succeeded")

	var validationError error
	var ok bool
	if ok, validationError = validateSecret(c, &createSecretRequest, validate, true); !ok {
		return
	}

	secretID, err := secret.RestrictedStore.Store(organizationID, &createSecretRequest)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Error during store"
		if secret.IsCASError(err) {
			statusCode = http.StatusConflict
			message = "Secret with this name already exists"
		} else {
			log.Errorf("Error during store: %s", err.Error())
		}
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	log.Infof("Secret stored at: %d/%s", organizationID, secretID)

	var errorMsg string
	if validationError != nil {
		errorMsg = validationError.Error()
	}

	s, err := secret.RestrictedStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, client.CreateSecretResponse{
		Name:      s.Name,
		Type:      s.Type,
		Id:        secretID,
		Error:     errorMsg,
		UpdatedAt: s.UpdatedAt,
		UpdatedBy: s.UpdatedBy,
		Version:   int32(s.Version),
		Tags:      s.Tags,
	})
}

// UpdateSecrets updates the given secret in Vault
func UpdateSecrets(c *gin.Context) {

	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Debugf("Organization id: %d", organizationID)

	secretID := getSecretID(c)

	validateParam := c.DefaultQuery("validate", "true")
	validate, err := strconv.ParseBool(validateParam)
	if err != nil {
		validate = true
	}

	log.Infof("validate value %t", validate)

	var createSecretRequest secret.CreateSecretRequest
	if err := c.ShouldBind(&createSecretRequest); err != nil {
		log.Errorf("Error during binding CreateSecretRequest: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during binding",
			Error:   err.Error(),
		})
		return
	}

	if createSecretRequest.Version == nil {
		msg := "Error during binding CreateSecretRequest: version can't be empty"
		log.Error(msg)
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during binding",
			Error:   msg,
		})
		return
	}

	createSecretRequest.UpdatedBy = auth.GetCurrentUser(c.Request).Login

	//Check if the received value is base64 encoded if not encode it.
	if createSecretRequest.Values[secretTypes.K8SConfig] != "" {
		createSecretRequest.Values[secretTypes.K8SConfig] = utils.EncodeStringToBase64(createSecretRequest.Values[secretTypes.K8SConfig])
	}

	log.Info("Binding request succeeded")

	var validationError error
	var ok bool
	if ok, validationError = validateSecret(c, &createSecretRequest, validate, false); !ok {
		return
	}

	if err := secret.RestrictedStore.Update(organizationID, secretID, &createSecretRequest); err != nil {
		statusCode := http.StatusInternalServerError
		if secret.IsCASError(err) {
			statusCode = http.StatusBadRequest
		}
		log.Errorf("Error during update: %s", err.Error())
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: "Error during update",
			Error:   err.Error(),
		})
		return
	}

	log.Debugf("Secret updated at: %s/%s", organizationID, secretID)

	s, err := secret.RestrictedStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	var errorMsg string
	if validationError != nil {
		errorMsg = validationError.Error()
	}

	c.JSON(http.StatusOK, client.CreateSecretResponse{
		Name:      s.Name,
		Type:      s.Type,
		Id:        secretID,
		Error:     errorMsg,
		UpdatedAt: s.UpdatedAt,
		UpdatedBy: s.UpdatedBy,
		Version:   int32(s.Version),
	})
}

// ListSecrets returns the user all secrets, if the secret type or tag is filled
// then a filtered response is returned
func ListSecrets(c *gin.Context) {

	organizationID := auth.GetCurrentOrganization(c.Request).ID

	var query secretTypes.ListSecretsQuery
	err := c.BindQuery(&query)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Failed to parse query",
			Error:   err.Error(),
		})
		return
	}

	log.Debugln("Organization:", organizationID, "type:", query.Type, "tags:", query.Tags, "values:", query.Values)

	if err := IsValidSecretType(query.Type); err != nil {
		log.Errorf("Error validation secret type[%s]: %s", query.Type, err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Not supported secret type",
			Error:   err.Error(),
		})
	} else {
		if secrets, err := secret.RestrictedStore.List(organizationID, &query); err != nil {
			log.Errorf("Error during listing secrets: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error during listing secrets",
				Error:   err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, secrets)
		}
	}
}

// GetSecret returns a secret by ID
func GetSecret(c *gin.Context) {

	organizationID := auth.GetCurrentOrganization(c.Request).ID

	secretID := getSecretID(c)

	if secret, err := secret.RestrictedStore.Get(organizationID, secretID); err != nil {
		log.Errorf("Error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing secret",
			Error:   err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, secret)
	}
}

// DeleteSecrets delete a secret with the given secret id
func DeleteSecrets(c *gin.Context) {
	log.Info("Start deleting secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id: %d", organizationID)

	secretID := getSecretID(c)

	log.Infof("Check clusters before delete secret[%s]", secretID)
	if err := checkClustersBeforeDelete(organizationID, secretID); err != nil {
		log.Errorf("Cluster found with this secret[%s]: %s", secretID, err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Cluster found with this secret[%s]", secretID),
			Error:   err.Error(),
		})
	} else if err := secret.RestrictedStore.Delete(organizationID, secretID); err != nil {
		log.Errorf("Error during deleting secrets: %s", err.Error())
		code := http.StatusInternalServerError
		resp := common.ErrorResponse{
			Code:    code,
			Message: "Error during deleting secrets",
			Error:   err.Error(),
		}
		c.AbortWithStatusJSON(code, resp)
	} else {
		log.Info("Delete secrets succeeded")
		c.Status(http.StatusNoContent)
	}
}

// GetSecretTags returns tags of a secret by ID
func GetSecretTags(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	secretID := getSecretID(c)
	log.Debugf("getting secret tags: %d/%s", organizationID, secretID)

	existingSecret, err := secret.RestrictedStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Error during getting secret",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, existingSecret.Tags)
}

// AddSecretTag adds a tag to a given secret in Vault
func AddSecretTag(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	secretID := getSecretID(c)
	tag := strings.Trim(c.Param("tag"), "/")
	log.Debugf("adding secret tag: %s to %d/%s", tag, organizationID, secretID)

	if tag == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Tag can not be empty",
		})
		return
	}

	if strings.HasPrefix(tag, "banzai:") {
		log.Errorf("error during secret tag add, restricted tag: %s", tag)
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Adding 'banzai:*' tag is restricted",
		})
		return
	}

	existingSecret, err := secret.RestrictedStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Error during getting secret",
			Error:   err.Error(),
		})
		return
	}

	createSecretRequest := secret.CreateSecretRequest{
		Name:      existingSecret.Name,
		Type:      existingSecret.Type,
		Values:    existingSecret.Values,
		Tags:      addElement(existingSecret.Tags, tag),
		Version:   &existingSecret.Version,
		UpdatedBy: auth.GetCurrentUser(c.Request).Login,
	}

	if err := secret.RestrictedStore.Update(organizationID, secretID, &createSecretRequest); err != nil {
		statusCode := http.StatusInternalServerError
		if secret.IsCASError(err) {
			statusCode = http.StatusBadRequest
		}

		log.Errorf("error during update: %s", err.Error())
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: "Error during update",
			Error:   err.Error(),
		})
		return
	}

	log.Debugf("added secret tag: %s to %d/%s", tag, organizationID, secretID)
	c.JSON(http.StatusOK, createSecretRequest.Tags)
}

// DeleteSecretTag removes a tag from a given secret in Vault
func DeleteSecretTag(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	secretID := getSecretID(c)
	tag := strings.Trim(c.Param("tag"), "/")
	log.Debugf("deleting secret tag: %s from %d/%s", tag, organizationID, secretID)

	if tag == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Tag can not be empty",
		})
		return
	}

	if strings.HasPrefix(tag, "banzai:") {
		log.Errorf("error during secret tag delete, restricted tag: %s", tag)
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Deleting 'banzai:*' tag is restricted",
		})
		return
	}

	existingSecret, err := secret.RestrictedStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusNotFound, common.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Error during getting secret",
			Error:   err.Error(),
		})
		return
	}

	createSecretRequest := secret.CreateSecretRequest{
		Name:      existingSecret.Name,
		Type:      existingSecret.Type,
		Values:    existingSecret.Values,
		Tags:      removeElement(existingSecret.Tags, tag),
		Version:   &existingSecret.Version,
		UpdatedBy: auth.GetCurrentUser(c.Request).Login,
	}

	if err := secret.RestrictedStore.Update(organizationID, secretID, &createSecretRequest); err != nil {
		statusCode := http.StatusInternalServerError
		if secret.IsCASError(err) {
			statusCode = http.StatusBadRequest
		}

		log.Errorf("error during update: %s", err.Error())
		c.AbortWithStatusJSON(statusCode, common.ErrorResponse{
			Code:    statusCode,
			Message: "Error during update",
			Error:   err.Error(),
		})
		return
	}

	log.Debugf("deleted secret tag: %s from %d/%s", tag, organizationID, secretID)
	c.Status(http.StatusNoContent)
}

// ListAllowedSecretTypes returns the allowed secret types and the required keys
func ListAllowedSecretTypes(c *gin.Context) {

	log.Info("Start listing allowed types and required keys")

	secretType := c.Param("type")
	log.Infof("Secret type: %s", secretType)

	if response, err := GetAllowedTypes(secretType); err != nil {
		log.Errorf("Error during listing allowed types: %s", err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing allowed types",
			Error:   err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, response)
	}
}

// GetAllowedTypes filters the allowed secret types if necessary
func GetAllowedTypes(secretType string) (interface{}, error) {
	if len(secretType) == 0 {
		log.Info("List all types and keys")
		return secretTypes.DefaultRules, nil
	} else if err := IsValidSecretType(secretType); err != nil {
		return nil, err
	} else {
		log.Info("Valid secret type. List filtered secret types")
		return secretTypes.DefaultRules[secretType], nil
	}
}

// IsValidSecretType checks the given secret type is supported
func IsValidSecretType(secretType string) error {
	if len(secretType) != 0 {
		if _, ok := secretTypes.DefaultRules[secretType]; !ok {
			return ErrNotSupportedSecretType
		}
	}
	return nil
}

// checkClustersBeforeDelete returns error if there's a running cluster that created with the given secret
func checkClustersBeforeDelete(orgId uint, secretId string) error {
	// TODO: move these to a struct and create them only once upon application init
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(intCluster.NewClusters(config.DB()), secretValidator, cluster.NewNopClusterEvents(), nil, nil, nil, log, errorHandler)

	clusters, err := clusterManager.GetClustersBySecretID(context.Background(), orgId, secretId)
	if err != nil {
		log.Warnf("could not get clusters: %s", err.Error())
	}

	if len(clusters) == 0 {
		log.Infof("no cluster found in database with the given orgId[%d] and secretId[%s]", orgId, secretId)
		return nil
	}

	for _, c := range clusters {
		if _, err := c.GetStatus(); err == nil {
			return fmt.Errorf("there's a running cluster with this secret: %s[%d]", c.GetName(), c.GetID())
		}
	}

	return nil
}

func addElement(s []string, v string) []string {
	for _, vv := range s {
		if vv == v {
			return s
		}
	}
	return append(s, v)
}

func removeElement(s []string, v string) []string {
	for i, vv := range s {
		if vv == v {
			s = append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func getSecretID(ctx *gin.Context) string {
	return ctx.Param("id")
}
