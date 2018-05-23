package api

import (
	"net/http"

	"encoding/base64"
	"fmt"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
)

// ErrNotSupportedSecretType describe an error if the secret type is not supported
var ErrNotSupportedSecretType = errors.New("Not supported secret type")

// AddSecrets saves the given secret to vault
func AddSecrets(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "Create Secrets"})
	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

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
	//Check if the received value is base64 encoded if not encode it.
	if createSecretRequest.Values[secret.K8SConfig] != "" {
		createSecretRequest.Values[secret.K8SConfig] = encodeStringToBase64(createSecretRequest.Values[secret.K8SConfig])
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
// if repo is set list the secrets for a given repo
func ListSecrets(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "List Secrets"})
	log.Info("Start listing secrets")

	log.Info("Get organization id and secret type from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	secretType := c.Query("type")
	repoName := c.Query("reponame")
	log.Infof("Organization id: %s", organizationID)
	log.Infof("Secret type: %s", secretType)
	log.Infof("Repository name: %s", repoName)

	if err := IsValidSecretType(secretType); err != nil {
		log.Errorf("Error validation secret type[%s]: %s", secretType, err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Not supported secret type",
			Error:   err.Error(),
		})
	} else {
		if items, err := secret.Store.List(organizationID, secretType, repoName, false); err != nil {
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
	log := logger.WithFields(logrus.Fields{"tag": "Delete Secrets"})
	log.Info("Start deleting secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).IDString()
	log.Infof("Organization id: %s", organizationID)

	secretID := c.Param("secretid")

	log.Infof("Check clusters before delete secret[%s]", secretID)
	if err := checkClustersBeforeDelete(organizationID, secretID); err != nil {
		log.Errorf("Cluster found with this secret[%s]: %s", secretID, err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Cluster found with this secret[%s]", secretID),
			Error:   err.Error(),
		})
	} else if err := secret.Store.Delete(organizationID, secretID); err != nil {
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
	log := logger.WithFields(logrus.Fields{"tag": "List allowed types/required keys"})

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
			return ErrNotSupportedSecretType
		}
	}
	return nil
}

// checkClustersBeforeDelete returns error if there's a running cluster that created with the given secret
func checkClustersBeforeDelete(orgId, secretId string) error {

	filter := map[string]interface{}{
		"organization_id": orgId,
		"secret_id":       secretId,
	}

	modelCluster, err := model.QueryCluster(filter)
	if err != nil {
		log.Infof("No cluster found in database with the given orgId[%s] and secretId[%s]", orgId, secretId)
		return nil
	}

	for _, mc := range modelCluster {
		if commonCluster, err := cluster.GetCommonClusterFromModel(&mc); err == nil {
			if _, err := commonCluster.GetStatus(); err == nil {
				return fmt.Errorf("there's a running cluster with this secret: %s[%d]", mc.Name, mc.ID)
			}
		}
	}
	return nil
}

// encodeStringToBase64 first checks if the string is encoded if yes returns it if no than encodes it.
func encodeStringToBase64(s string) string {
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}
	return s
}
