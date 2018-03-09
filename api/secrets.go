package api

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/gin-gonic/gin"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var secretStoreObj *secretStore

func init() {
	secretStoreObj = newVaultSecretStore()
}

func AddSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "Create Secrets"})
	log.Info("Start adding secrets")

	log.Info("Get organization id from params")
	organizationId := c.Param("id")
	log.Infof("Organization id: %s", organizationId)

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
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Validation error",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Validation passed")

	// org/{org_id}/{uuid}/{secret_type}
	secretId := generateSecretId()
	vaultPath := fmt.Sprintf("org/%s/%s/%s", organizationId, secretId, createSecretRequest.SecretType)
	if err := secretStoreObj.mount(vaultPath, createSecretRequest.Name); err != nil {
		log.Errorf("Error during mount: %s", err.Error())
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during mount",
			Error:   err.Error(),
		})
	}

	log.Info("Mount succeeded")

	for _, kv := range createSecretRequest.Values {
		path := fmt.Sprintf("%s/%s", vaultPath, kv.Key)
		if err := secretStoreObj.Store(path, kv.Value); err != nil {
			log.Errorf("Error during store: %s", err.Error())
			c.JSON(http.StatusInternalServerError, components.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during store",
				Error:   err.Error(),
			})
		} else {
			log.Infof("Secret stored: %s", path)
		}
	}

	c.JSON(http.StatusCreated, CreateSecretResponse{
		Name:       createSecretRequest.Name,
		SecretType: createSecretRequest.SecretType,
		SecretId:   secretId,
	})

}

func ListSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "List Secrets"})
	log.Info("Start listing secrets")

	log.Info("Get organization id from params")
	organizationId := c.Param("id")
	log.Infof("Organization id: %s", organizationId)

	if items, err := secretStoreObj.List(organizationId); err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
	organizationId := c.Param("id")
	secretId := c.Param("secretId")
	log.Infof("Organization id: %s", organizationId)

	if err := secretStoreObj.delete(organizationId, secretId); err != nil {
		log.Errorf("Error during deleting secrets: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during deleting secrets",
			Error:   err.Error(),
		})
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
	client       *vault.Client
	logical      *vault.Logical
	tokenRenewer *vault.Renewer
}

func (ss *secretStore) Store(path string, value string) error {
	log.Infof("Start storing secret")
	data := map[string]interface{}{"value": value}
	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during store secrets")
	}
	return nil
}

func (ss *secretStore) List(organizationId string) ([]SecretsItemResponse, error) {

	log.Info("Listing mounts")
	if mounts, err := secretStoreObj.client.Sys().ListMounts(); err != nil {
		return nil, err
	} else {
		var responseItems []SecretsItemResponse
		for _, secretType := range allSecretTypes {
			for key, mount := range mounts {
				// find mount
				log.Debugf("Searching for organization mounts [%s]", secretType)
				prefix := fmt.Sprintf("org/%s", organizationId)
				suffix := fmt.Sprintf("/%s/", secretType)
				if strings.HasPrefix(key, fmt.Sprintf("org/%s", organizationId)) && strings.HasSuffix(key, suffix) {

					desc := mount.Description

					secretId := key[len(prefix)+1 : len(key)-len(suffix)]
					log.Debugf("Secret id: %s", secretId)

					sir := SecretsItemResponse{
						Id:         secretId,
						Name:       desc,
						SecretType: string(secretType),
						Values:     nil,
					}

					if secret, err := secretStoreObj.client.Logical().List(key); err != nil {
						log.Errorf("Error listing secrets: %s", err.Error())
					} else {
						keys := secret.Data["keys"].([]interface{})
						var secrets []KeyValue
						for _, key := range keys {
							secrets = append(secrets, KeyValue{
								Key: key.(string),
							})
						}
						sir.Values = secrets
						responseItems = append(responseItems, sir)
					}

				}
			}
		}

		return responseItems, nil
	}
}

func newVaultSecretStore() *secretStore {
	client, err := vault.NewClient(vault.DefaultConfig())
	if err != nil {
		panic(err)
	}
	logical := client.Logical()
	var tokenRenewer *vault.Renewer

	if client.Token() == "" {

		tokenPath := viper.GetString("auth.vaultpath")
		token, err := ioutil.ReadFile(tokenPath + "/.vault-token")
		if err == nil {

			client.SetToken(string(token))

		} else {
			// If VAULT_TOKEN or ~/.vault-token wasn't provided let's suppose
			// we are in Kubernetes and try to get one with the ServiceAccount token

			k8sconfig, err := rest.InClusterConfig()
			if err != nil {
				panic(err)
			}

			data := map[string]interface{}{"jwt": k8sconfig.BearerToken, "role": "pipeline"}
			secret, err := logical.Write("auth/kubernetes/login", data)
			if err != nil {
				panic(err)
			}

			tokenRenewer, err = client.NewRenewer(&vault.RenewerInput{Secret: secret})
			if err != nil {
				panic(err)
			}

			// We never really want to stop this
			go tokenRenewer.Renew()

			// Finally set the first token from the response
			client.SetToken(secret.Auth.ClientToken)
		}
	}

	return &secretStore{client: client, logical: logical, tokenRenewer: tokenRenewer}
}

func (ss *secretStore) mount(mountPath, name string) error {

	log.Infof("Mount %s", mountPath)

	mountInput := &vault.MountInput{
		Type:        "kv",
		Description: name,
	}

	if err := ss.client.Sys().Mount(mountPath, mountInput); err != nil && !strings.Contains(err.Error(), "existing mount") {
		return errors.Wrap(err, "Error enabling")
	}
	return nil
}

func (ss *secretStore) delete(organizationId, secretId string) error {

	log.Info("Listing mounts")
	if mounts, err := secretStoreObj.client.Sys().ListMounts(); err != nil {
		return err
	} else {
		for key := range mounts {

			prefix := fmt.Sprintf("org/%s/%s/", organizationId, secretId)
			if strings.HasPrefix(key, prefix) {
				return ss.client.Sys().Unmount(key)
			}

		}
	}

	return errors.New(fmt.Sprintf("There are no secrets with [%s] organization id and [%s] secret id", organizationId, secretId))
}

func generateSecretId() string {
	log.Debug("Generate secret id")
	rInt := rand.Intn(10)
	return fmt.Sprintf("%d%d", time.Now().UTC().Unix(), rInt)
}

type SecretType string

var allSecretTypes = []SecretType{
	Amazon,
	Azure,
	// Google, // todo put back if the rules are completed
}

func getRules() []rule {
	// todo add google rules
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
	// Google SecretType = "GOOGLE_SECRET" // todo put back if the rules are completed
)
