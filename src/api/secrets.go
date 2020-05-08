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

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/secret/restricted"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

// ValidateSecret validates the given secret
func ValidateSecret(c *gin.Context) {
	log.Info("start validation secret")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id [%d]", organizationID)

	secretID := getSecretID(c)
	log.Infof("secret id [%d]", secretID)

	err := restricted.GlobalSecretStore.Verify(organizationID, secretID)
	if err != nil {
		log.Errorf("Error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting secret",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
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

	log.Info("Binding request succeeded")

	createSecretRequest.Verify = validate

	secretID, err := restricted.GlobalSecretStore.Store(organizationID, &createSecretRequest)
	if err != nil {
		var verr interface {
			Validation() bool
		}

		if errors.As(err, &verr) && verr.Validation() {
			c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Validation error",
				Error:   err.Error(),
			})

			return
		}

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

	s, err := restricted.GlobalSecretStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, pipeline.CreateSecretResponse{
		Name:      s.Name,
		Type:      s.Type,
		Id:        secretID,
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

	createSecretRequest.UpdatedBy = auth.GetCurrentUser(c.Request).Login

	log.Info("Binding request succeeded")

	createSecretRequest.Verify = validate

	if err := restricted.GlobalSecretStore.Update(organizationID, secretID, &createSecretRequest); err != nil {
		var verr interface {
			Validation() bool
		}

		if errors.As(err, &verr) && verr.Validation() {
			c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Validation error",
				Error:   err.Error(),
			})

			return
		}

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

	s, err := restricted.GlobalSecretStore.Get(organizationID, secretID)
	if err != nil {
		log.Errorf("error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, pipeline.CreateSecretResponse{
		Name:      s.Name,
		Type:      s.Type,
		Id:        secretID,
		UpdatedAt: s.UpdatedAt,
		UpdatedBy: s.UpdatedBy,
		Version:   int32(s.Version),
		Tags:      s.Tags,
	})
}

// ListSecrets returns the user all secrets, if the secret type or tag is filled
// then a filtered response is returned
func ListSecrets(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	var query secret.ListSecretsQuery
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

	if secrets, err := restricted.GlobalSecretStore.List(organizationID, &query); err != nil {
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

// GetSecret returns a secret by ID
func GetSecret(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	secretID := getSecretID(c)

	if s, err := restricted.GlobalSecretStore.Get(organizationID, secretID); err != nil {
		status := http.StatusBadRequest

		if errors.Is(err, secret.ErrSecretNotExists) {
			status = http.StatusNotFound
		}

		log.Errorf("Error during getting secret: %s", err.Error())
		c.AbortWithStatusJSON(status, common.ErrorResponse{
			Code:    status,
			Message: "Error during listing secret",
			Error:   err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, s)
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
	} else if err := restricted.GlobalSecretStore.Delete(organizationID, secretID); err != nil {
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

	existingSecret, err := restricted.GlobalSecretStore.Get(organizationID, secretID)
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

	existingSecret, err := restricted.GlobalSecretStore.Get(organizationID, secretID)
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
		UpdatedBy: auth.GetCurrentUser(c.Request).Login,
	}

	if err := restricted.GlobalSecretStore.Update(organizationID, secretID, &createSecretRequest); err != nil {
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

	existingSecret, err := restricted.GlobalSecretStore.Get(organizationID, secretID)
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
		UpdatedBy: auth.GetCurrentUser(c.Request).Login,
	}

	if err := restricted.GlobalSecretStore.Update(organizationID, secretID, &createSecretRequest); err != nil {
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

// checkClustersBeforeDelete returns error if there's a running cluster that created with the given secret
func checkClustersBeforeDelete(orgId uint, secretId string) error {
	// TODO: move these to a struct and create them only once upon application init
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterRepo := clusteradapter.NewClusters(global.DB())
	clusterStore := clusteradapter.NewStore(global.DB(), clusterRepo)
	clusterManager := cluster.NewManager(clusterRepo, secretValidator, cluster.NewNopClusterEvents(), nil, nil, nil, log, errorHandler, clusterStore, nil)

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
