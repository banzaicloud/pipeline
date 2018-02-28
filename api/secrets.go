package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"github.com/banzaicloud/banzai-types/components"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"io/ioutil"
	"os/user"
	"k8s.io/client-go/rest"
	"strings"
	"fmt"
	"time"
	"math/rand"
)

var secretStoreObj *secretStore

func init() {
	secretStoreObj = newVaultSecretStore()
}

func AddSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "Create Secrets"})
	log.Info("Start adding secrets")

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

	// todo validate types

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

}

func ListSecrets(c *gin.Context) {

	log = logger.WithFields(logrus.Fields{"tag": "List Secrets"})
	log.Info("Start listing secrets")

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
		log.Infof("Listing secrets succeeded: %v", items)
		c.JSON(http.StatusOK, ListSecretsResponse{
			Secrets: items,
		})
	}

}

type CreateSecretRequest struct {
	Name       string     `json:"name" binding:"required"`
	SecretType string     `json:"type" binding:"required"`
	Values     []KeyValue `json:"values" binding:"required"`
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
	data := map[string]interface{}{"value": value}
	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during store secrets")
	}
	return nil
}

func (ss *secretStore) List(organizationId string) ([]SecretsItemResponse, error) {

	log.Info("List mounts")
	if mounts, err := secretStoreObj.client.Sys().ListMounts(); err != nil {
		return nil, err
	} else {
		var responseItems []SecretsItemResponse
		for _, secretType := range allSecretTypes {
			for key, mount := range mounts {
				// find mount
				log.Debugf("Searching for organization mounts [%s]", secretType)
				prefix := fmt.Sprintf("org/%s", organizationId)
				suffix := fmt.Sprintf("%s/", secretType)
				if strings.HasPrefix(key, fmt.Sprintf("org/%s", organizationId)) && strings.HasSuffix(key, suffix) {

					desc := mount.Description

					secretId := key[len(prefix)+1:len(suffix)]
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

		usr, err := user.Current()
		if err != nil {
			panic(err)
		}

		token, err := ioutil.ReadFile(usr.HomeDir + "/.vault-token")
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

	mountInput := &vault.MountInput{
		Type:        "kv",
		Description: name,
	}

	if err := ss.client.Sys().Mount(mountPath, mountInput); err != nil && !strings.Contains(err.Error(), "existing mount") {
		return errors.Wrap(err, "Error enabling")
	}
	return nil
}

func generateSecretId() string {
	rInt := rand.Intn(10)
	return fmt.Sprintf("%d%d", time.Now().UTC().Unix(), rInt)
}

type SecretType string

var allSecretTypes = []SecretType{
	Amazon,
	Azure,
	Google,
}

const (
	Amazon SecretType = "AMAZON_SECRET"
	Azure  SecretType = "AZURE_SECRET"
	Google SecretType = "GOOGLE_SECRET"
)
