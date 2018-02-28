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

	// org/{org_id}/{type}
	vaultPath := fmt.Sprintf("org/%s/%s", organizationId, createSecretRequest.SecretType)
	if err := secretStoreObj.mount(vaultPath, createSecretRequest.Name); err != nil {
		log.Errorf("Error during mount: %s", err.Error())
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during mount",
			Error:   err.Error(),
		})
	}

	secretId := generateSecretId()
	for _, kv := range createSecretRequest.Values {
		path := fmt.Sprintf("%s/%s/%s", vaultPath, secretId, kv.Key)
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

	var allSecrets []SecretsItemResponse

	for _, secretType := range allSecretTypes {
		// get secrets on path
		path := fmt.Sprintf("org/%s/%s", organizationId, secretType)
		log.Infof("Listing secrets on %s", path)
		if secrets, err := secretStoreObj.List(path); err != nil {
			log.Errorf("Error during listing secret keys: %s", err.Error())
		} else {
			log.Info("Listing secrets on path succeeded")

			var values []KeyValue
			for _, key := range secrets.keys {
				values = append(values, KeyValue{
					Key: key,
				})
			}

			sir := SecretsItemResponse{
				Id:         strings.Replace(secrets.id, "/", "", -1),
				Name:       secrets.description,
				SecretType: string(secretType),
				Values:     values,
			}
			allSecrets = append(allSecrets, sir)
		}
	}

	c.JSON(http.StatusOK, ListSecretsResponse{
		Secrets: allSecrets,
	})

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

func (ss *secretStore) List(path string) (*secretListItem, error) {
	var result secretListItem

	log.Info("List mounts")
	if mounts, err := ss.client.Sys().ListMounts(); err != nil {
		return nil, errors.Wrap(err, "Error during listing mounts")
	} else {
		mount, ok := mounts[fmt.Sprintf("%s/", path)]
		if !ok {
			return nil, errors.New(fmt.Sprintf("No mount found with name: %s", path))
		}

		result.description = mount.Description

		if secret, err := ss.client.Logical().List(path); err != nil {
			return nil, errors.Wrap(err, "Error during getting secret id")
		} else if secret != nil {
			keys := secret.Data["keys"].([]interface{})
			for _, secretId := range keys {
				path = fmt.Sprintf("%s/%s", path, secretId.(string))
				if secret, err := ss.client.Logical().List(path); err != nil {
					return nil, errors.Wrap(err, "Error during list secret keys")
				} else if secret != nil {
					keys := secret.Data["keys"].([]interface{})
					var secrets []string
					for _, key := range keys {
						secrets = append(secrets, key.(string))
					}

					result.id = secretId.(string)
					result.keys = secrets

					return &result, nil
				}
			}
		}
	}

	return nil, errors.New(fmt.Sprintf("There are no secrets on %s path", path))
}

type secretListItem struct {
	id          string
	keys        []string
	description string
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
