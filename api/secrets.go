package api

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/common"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
)

// ErrNotSupportedSecretType describe an error if the secret type is not supported
var ErrNotSupportedSecretType = errors.New("Not supported secret type")

// AddSecrets saves the given secret to vault
func AddSecrets(c *gin.Context) {

	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id: %d", organizationID)

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
	//Check if the received value is base64 encoded if not encode it.
	if createSecretRequest.Values[secretTypes.K8SConfig] != "" {
		createSecretRequest.Values[secretTypes.K8SConfig] = utils.EncodeStringToBase64(createSecretRequest.Values[secretTypes.K8SConfig])
	}

	log.Info("Binding request succeeded")

	log.Info("Start validation")
	verifier := verify.NewVerifier(createSecretRequest.Type, createSecretRequest.Values)
	if err := createSecretRequest.Validate(verifier); err != nil {
		log.Errorf("Validation error: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Validation error",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Validation passed")

	secretID, err := secret.Store.Store(organizationID, &createSecretRequest)
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

	c.JSON(http.StatusCreated, secret.CreateSecretResponse{
		Name: createSecretRequest.Name,
		Type: createSecretRequest.Type,
		ID:   secretID,
	})
}

// UpdateSecrets update the given secret to vault
func UpdateSecrets(c *gin.Context) {

	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Debugf("Organization id: %d", organizationID)

	secretID := c.Param("secretid")

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

	//Check if the received value is base64 encoded if not encode it.
	if createSecretRequest.Values[secretTypes.K8SConfig] != "" {
		createSecretRequest.Values[secretTypes.K8SConfig] = utils.EncodeStringToBase64(createSecretRequest.Values[secretTypes.K8SConfig])
	}

	log.Info("Binding request succeeded")

	log.Info("Start validation")
	verifier := verify.NewVerifier(createSecretRequest.Type, createSecretRequest.Values)
	if err := createSecretRequest.Validate(verifier); err != nil {
		log.Errorf("Validation error: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Validation error",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Validation passed")

	if err := secret.Store.Update(organizationID, secretID, &createSecretRequest); err != nil {
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

	c.JSON(http.StatusOK, secret.CreateSecretResponse{
		Name: createSecretRequest.Name,
		Type: createSecretRequest.Type,
		ID:   secretID,
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

	log.Debugln("Organization:", organizationID, "type:", query.Type, "tag:", query.Tag, "values:", query.Values)

	if err := IsValidSecretType(query.Type); err != nil {
		log.Errorf("Error validation secret type[%s]: %s", query.Tag, err.Error())
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Not supported secret type",
			Error:   err.Error(),
		})
	} else {
		if secrets, err := secret.Store.List(organizationID, &query); err != nil {
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

// DeleteSecrets delete a secret with the given secret id
func DeleteSecrets(c *gin.Context) {
	log.Info("Start deleting secrets")

	log.Info("Get organization id from params")
	organizationID := auth.GetCurrentOrganization(c.Request).ID
	log.Infof("Organization id: %d", organizationID)

	secretID := c.Param("secretid")

	log.Infof("Check clusters before delete secret[%s]", secretID)
	if err := checkClustersBeforeDelete(organizationID, secretID); err != nil {
		log.Errorf("Cluster found with this secret[%s]: %s", secretID, err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, common.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Cluster found with this secret[%s]", secretID),
			Error:   err.Error(),
		})
	} else if err := searchForbiddenTags(organizationID, secretID); err != nil {
		log.Errorf("Error during deleting secrets: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during deleting secrets",
			Error:   err.Error(),
		})
	} else if err := secret.Store.Delete(organizationID, secretID); err != nil {
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
		return secret.AllowedSecretTypesResponse{
			Allowed: secretTypes.DefaultRules,
		}, nil
	} else if err := IsValidSecretType(secretType); err != nil {
		return nil, err
	} else {
		log.Info("Valid secret type. List filtered secret types")
		return secret.AllowedFilteredSecretTypesResponse{
			Keys: secretTypes.DefaultRules[secretType],
		}, nil
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

	filter := map[string]interface{}{
		"organization_id": orgId,
		"secret_id":       secretId,
	}

	modelCluster, err := model.QueryCluster(filter)
	if err != nil {
		log.Infof("No cluster found in database with the given orgId[%d] and secretId[%s]", orgId, secretId)
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

// searchForbiddenTags gets the secret by organization id and secret id and looks for forbidden tag(s)
// Secrets cannot be created/deleted with these tags
func searchForbiddenTags(orgId uint, secretId string) error {

	secretItem, err := secret.Store.Get(orgId, secretId)
	if err != nil {
		return err
	}

	return secret.IsForbiddenTag(secretItem.Tags)
}
