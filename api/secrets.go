package api

import (
	"net/http"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/go-errors/errors"
)

var NotSupportedSecretType = errors.New("Not supported secret type")

// AddSecrets saves the given secret to vault
func AddSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "Create Secrets"})
	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	log.Info("Binding request")

	var createSecretRequest secret.CreateSecretRequest
	if err := c.ShouldBind(&createSecretRequest); err != nil {
		log.Errorf("Error during binding CreateSecretRequest: %s", err.Error())
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

	secretID := secret.GenerateSecretID()
	if err := secret.Store.Store(organizationID, secretID, createSecretRequest); err != nil {
		log.Errorf("Error during store: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during store",
			Error:   err.Error(),
		})
		return
	}

	log.Infof("Secret stored at: %s/%s", organizationID, secretID)

	c.JSON(http.StatusCreated, secret.CreateSecretResponse{
		Name:       createSecretRequest.Name,
		SecretType: createSecretRequest.SecretType,
		SecretID:   secretID,
	})
}

// ListSecrets returns the user all secrets, if the secret type is filled, then filtered
func ListSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "List Secrets"})
	log.Info("Start listing secrets")

	log.Info("Get organization id and secret type from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	secretType := c.Param("type")
	log.Infof("Organization id: %s", organizationID)
	log.Infof("Secret type: %s", secretType)

	if err := IsValidSecretType(secretType); err != nil {
		log.Errorf("Error validation secret type[%s]: %s", secretType, err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Not supported secret type",
			Error:   err.Error(),
		})
	} else {
		if items, err := secret.Store.List(organizationID, secretType); err != nil {
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
}

// DeleteSecrets delete a secret with the given secret id
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

// ListAllowedSecretTypes returns the allowed secret types and the required keys
func ListAllowedSecretTypes(c *gin.Context) {
	log = logger.WithFields(logrus.Fields{"tag": "List allowed types/required keys"})

	log.Info("Start listing allowed types and required keys")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	secretType := c.Param("type")
	log.Infof("Secret type: %s", secretType)

	if response, err := GetAllowedTypes(secretType); err != nil {
		log.Errorf("Error during listing allowed types: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
		return secret.AllowedSecretTypesResponse{
			Allowed: secret.DefaultRules,
		}, nil
	} else if err := IsValidSecretType(secretType); err != nil {
		return nil, err
	} else {
		log.Info("Valid secret type. List filtered secret types")
		return secret.AllowedFilteredSecretTypesResponse{
			Keys: secret.DefaultRules[secretType],
		}, nil
	}
}

// IsValidSecretType checks the given secret type is supported
func IsValidSecretType(secretType string) error {
	if len(secretType) != 0 {
		r := secret.DefaultRules[secretType]
		if r == nil {
			return NotSupportedSecretType
		}
	}
	return nil
}
