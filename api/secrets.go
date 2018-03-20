package api

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func AddSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "Create Secrets"})
	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	log.Info("Binding request")

	var createSecretRequest secret.CreateSecretRequest
	if err := c.ShouldBind(&createSecretRequest); err != nil {
		log.Errorf("Error during binding CreateSecretRequest: %s", err)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during binding",
			Error:   err.Error(),
		})
		return
	}

	log.Info("Binding request succeeded")
	log.Debugf("%#v", createSecretRequest)

	log.Info("Start validation")
	if err := createSecretRequest.Validate(); err != nil {
		log.Errorf("Validation error: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Validation error",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Validation passed")

	// orgs/{org_id}/{uuid}/{secret_type}
	secretID := secret.GenerateSecretID()
	secretPath := fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)

	if err := secret.Store.Store(secretPath, createSecretRequest); err != nil {
		log.Errorf("Error during store: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during store",
			Error:   err.Error(),
		})
		return
	}

	log.Infof("Secret stored at: %s", secretPath)

	c.JSON(http.StatusCreated, secret.CreateSecretResponse{
		Name:       createSecretRequest.Name,
		SecretType: createSecretRequest.SecretType,
		SecretID:   secretID,
	})
}

func ListSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "List Secrets"})
	log.Info("Start listing secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	if items, err := secret.Store.List(organizationID); err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing secrets",
			Error:   err.Error(),
		})
	} else {
		log.Infof("Listing secrets succeeded: %#v", items)
		c.JSON(http.StatusOK, secret.ListSecretsResponse{
			Secrets: items,
		})
	}
}

func DeleteSecrets(c *gin.Context) {
	log = logger.WithFields(logrus.Fields{"tag": "Delete Secrets"})
	log.Info("Start deleting secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	secretID := c.Param("secretid")

	if err := secret.Store.Delete(organizationID, secretID); err != nil {
		log.Errorf("Error during deleting secrets: %s", err.Error())
		code := http.StatusInternalServerError
		resp := components.ErrorResponse{
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
