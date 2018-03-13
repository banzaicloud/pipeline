package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/banzaicloud/bank-vaults/vault"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/gin-gonic/gin"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var secretStoreObj *secretStore

func init() {
	secretStoreObj = newVaultSecretStore()
}

func AddSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "Create Secrets"})
	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationID := strconv.FormatUint(uint64(auth.GetCurrentOrganization(c.Request).ID), 10)
	log.Infof("Organization id: %s", organizationID)

	log.Info("Binding request")

	var createSecretRequest CreateSecretRequest
	if err := c.Bind(&createSecretRequest); err != nil {
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
	secretID := generateSecretID()
	secretPath := fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)

	if err := secretStoreObj.store(secretPath, createSecretRequest); err != nil {
		log.Errorf("Error during store: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during store",
			Error:   err.Error(),
		})
		return
	}

	log.Infof("Secret stored at: %s", secretPath)

	c.JSON(http.StatusCreated, CreateSecretResponse{
		Name:       createSecretRequest.Name,
		SecretType: createSecretRequest.SecretType,
		SecretId:   secretID,
	})
}

func ListSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "List Secrets"})
	log.Info("Start listing secrets")

	log.Info("Get organization id from params")
	organizationId := strconv.FormatUint(uint64(c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID), 10)

	log.Infof("Organization id: %s", organizationId)

	if items, err := secretStoreObj.list(organizationId); err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing secrets",
			Error:   err.Error(),
		})
	} else {
		log.Infof("Listing secrets succeeded: %#v", items)
		c.JSON(http.StatusOK, ListSecretsResponse{
			Secrets: items,
		})
	}

}

func DeleteSecrets(c *gin.Context) {
	log = logger.WithFields(logrus.Fields{"tag": "Delete Secrets"})
	log.Info("Start deleting secrets")

	log.Info("Get organization id and secret id from params")
	organizationId := strconv.FormatUint(uint64(c.Request.Context().Value(auth.CurrentOrganization).(*auth.Organization).ID), 10)
	secretId := c.Param("secretId")
	log.Infof("Organization id: %s", organizationId)

	if err := secretStoreObj.delete(organizationId, secretId); err != nil {
		log.Errorf("Error during deleting secrets: %s", err.Error())
		isNotFound := strings.Contains(err.Error(), "There are no secrets with")
		msg := "Error during deleting secrets"
		code := http.StatusBadRequest
		if isNotFound {
			code = http.StatusNotFound
			msg = "Secrets not found"
		}

		resp := components.ErrorResponse{
			Code:    code,
			Message: msg,
			Error:   err.Error(),
		}

		c.JSON(code, resp)
	} else {
		log.Info("Delete secrets succeeded")
		c.Status(http.StatusOK)
	}

}

type CreateSecretResponse struct {
	Name       string `json:"name" binding:"required"`
	SecretType string `json:"type" binding:"required"`
	SecretId   string `json:"secret_id"`
}

type CreateSecretRequest struct {
	Name       string     `json:"name" binding:"required"`
	SecretType string     `json:"type" binding:"required"`
	Values     []KeyValue `json:"values" binding:"required"`
}

func (c *CreateSecretRequest) Validate() error {

	allRules := getRules()
	for i, rule := range allRules {
		if string(rule.secretType) == c.SecretType {
			for j, requiredKey := range rule.requiredKeys {
				for _, keyValues := range c.Values {
					if requiredKey.requiredKey == keyValues.Key {
						allRules[i].requiredKeys[j].isInRequest = true
						break
					}
				}
			}

			if err := allRules[i].isValid(); err != nil {
				return err
			}

			return nil
		}
	}

	return errors.New("Wrong secret type")
}

type KeyValue struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value,omitempty" binding:"required"`
}

type ListSecretsResponse struct {
	Secrets []SecretsItemResponse `json:"secrets"`
}

type SecretsItemResponse struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"`
	SecretType string     `json:"type"`
	Values     []KeyValue `json:"values"`
}

type secretStore struct {
	client  *vault.Client
	logical *vaultapi.Logical
}

func (ss *secretStore) store(path string, value CreateSecretRequest) error {
	log.Infof("Start storing secret")
	_, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "Error during storing secret")
	}
	data := map[string]interface{}{"value": value}
	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during storing secret")
	}
	return nil
}

func (ss *secretStore) list(organizationId string) ([]SecretsItemResponse, error) {

	log.Info("Listing secrets")
	responseItems := make([]SecretsItemResponse, 0)

	log.Debugf("Searching for organizations secrets [%s]", organizationId)
	orgSecretPath := fmt.Sprintf("secret/orgs/%s", organizationId)

	if secret, err := ss.logical.List(orgSecretPath); err != nil {
		log.Errorf("Error listing secrets: %s", err.Error())
	} else if secret != nil {
		keys := secret.Data["keys"].([]interface{})
		for _, key := range keys {
			secretID := key.(string)
			if secret, err := ss.logical.Read(orgSecretPath + "/" + secretID); err != nil {
				log.Errorf("Error listing secrets: %s", err.Error())
			} else if secret != nil {
				secretData := secret.Data["value"].(map[string]interface{})
				sir := SecretsItemResponse{
					Id:         key.(string),
					Name:       secretData["name"].(string),
					SecretType: secretData["type"].(string),
					Values:     nil,
				}
				responseItems = append(responseItems, sir)
			}
		}
	} else {
		return responseItems, nil
	}

	return responseItems, nil
}

func newVaultSecretStore() *secretStore {
	role := "pipeline"
	client, err := vault.NewClient(role)
	if err != nil {
		panic(err)
	}
	logical := client.Vault().Logical()
	return &secretStore{client: client, logical: logical}
}

func (ss *secretStore) delete(organizationId, secretId string) error {
	_, err := ss.logical.Delete(fmt.Sprintf("secret/orgs/%s/%s", organizationId, secretId))
	return err
}

func generateSecretID() string {
	log.Debug("Generating secret id")
	return uuid.NewV4().String()
}

type SecretType string

var allSecretTypes = []SecretType{
	Amazon,
	Azure,
	Google,
}

func getRules() []rule {
	return []rule{
		{
			secretType: Amazon,
			requiredKeys: []ruleKey{
				{requiredKey: "AWS_ACCESS_KEY_ID"},
				{requiredKey: "AWS_SECRET_ACCESS_KEY"},
			},
		},
		{
			secretType: Azure,
			requiredKeys: []ruleKey{
				{requiredKey: "AZURE_CLIENT_ID"},
				{requiredKey: "AZURE_CLIENT_SECRET"},
				{requiredKey: "AZURE_TENANT_ID"},
				{requiredKey: "AZURE_SUBSCRIPTION_ID"},
			},
		},
		{
			secretType: Google,
			requiredKeys: []ruleKey{
				{requiredKey: "CLIENT_ID"},
				{requiredKey: "CLIENT_SECRET"},
				{requiredKey: "REFRESH_TOKEN"},
				{requiredKey: "TYPE"},
			},
		},
	}
}

type rule struct {
	secretType   SecretType
	requiredKeys []ruleKey
}

func (r *rule) isValid() error {
	for _, ruleKey := range r.requiredKeys {
		if !ruleKey.isInRequest {
			return errors.New(fmt.Sprintf("Missing key: %s", ruleKey.requiredKey))
		}
	}
	return nil
}

type ruleKey struct {
	requiredKey string
	isInRequest bool
}

const (
	Amazon SecretType = "AMAZON_SECRET"
	Azure  SecretType = "AZURE_SECRET"
	Google SecretType = "GOOGLE_SECRET"
)
