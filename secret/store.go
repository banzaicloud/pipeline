package secret

import (
	"fmt"
	"github.com/banzaicloud/bank-vaults/vault"
	"github.com/banzaicloud/pipeline/config"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger
var Store *SecretStore

const (
	Amazon     = "AMAZON_SECRET"
	Azure      = "AZURE_SECRET"
	Google     = "GOOGLE_SECRET"
	General    = "GENERAL_SECRET"
	Kubernetes = "KUBERNETES_SECRET"
)

func init() {
	logger = config.Logger()
	Store = newVaultSecretStore()
}

type SecretStore struct {
	client  *vault.Client
	logical *vaultapi.Logical
}

type CreateSecretResponse struct {
	Name       string `json:"name" binding:"required"`
	SecretType string `json:"type" binding:"required"`
	SecretID   string `json:"secret_id"`
}

type CreateSecretRequest struct {
	Name       string            `json:"name" binding:"required"`
	SecretType string            `json:"type" binding:"required"`
	Values     map[string]string `json:"values" binding:"required"`
}

type ListSecretsResponse struct {
	Secrets []SecretsItemResponse `json:"secrets"`
}

type SecretsItemResponse struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	SecretType string            `json:"type"`
	Values     map[string]string `json:"-"`
}

func newVaultSecretStore() *SecretStore {
	role := "pipeline"
	client, err := vault.NewClient(role)
	if err != nil {
		panic(err)
	}
	logical := client.Vault().Logical()
	return &SecretStore{client: client, logical: logical}
}

func GenerateSecretID() string {
	log := logger.WithFields(logrus.Fields{"tag": "Secret"})
	log.Debug("Generating secret id")
	return uuid.NewV4().String()
}

var DefaultRules = map[string][]string{
	Amazon: {
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
	},
	Azure: {
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
		"AZURE_SUBSCRIPTION_ID",
	},
	Google: {
		"TYPE",
		"PROJECT_ID",
		"PRIVATE_KEY_ID",
		"PRIVATE_KEY",
		"CLIENT_EMAIL",
		"CLIENT_ID",
		"AUTH_URI",
		"TOKEN_URI",
		"AUTH_PROVIDER_X509_CERT_URL",
		"CLIENT_X509_CERT_URL",
	},
}

func (c *CreateSecretRequest) Validate() error {
	requiresKeys, ok := DefaultRules[c.SecretType]
	if !ok {
		return errors.Errorf("wrong secret type: %s", c.SecretType)
	}
	for _, key := range requiresKeys {
		if _, ok := c.Values[key]; !ok {
			return errors.Errorf("missing key: %s", key)
		}

	}
	return nil
}

func (ss *SecretStore) Delete(organizationID, secretID string) error {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteSecret"})
	log.Debugf("Delete sectret: %s", fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID))
	_, err := ss.logical.Delete(fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID))
	return err
}

func (ss *SecretStore) Store(path string, value CreateSecretRequest) error {
	log := logger.WithFields(logrus.Fields{"tag": "StoreSecret"})
	log.Infof("Start storing secret")
	data := map[string]interface{}{"value": value}
	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during storing secret")
	}
	return nil
}

func (ss *SecretStore) Get(organizationID string, secretID string) (*SecretsItemResponse, error) {
	secretPath := fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)
	secret, err := ss.logical.Read(secretPath)
	if err != nil {
		return nil, err
	}
	fmt.Println(organizationID)
	fmt.Println(secretID)
	fmt.Printf("%#v", secret)
	secretData := secret.Data["value"].(map[string]interface{})

	secretResp := &SecretsItemResponse{
		ID:         secretID,
		Name:       secretData["name"].(string),
		SecretType: secretData["type"].(string),
	}
	// Assert map[string]string to map[string]interface{}
	parsedValues := make(map[string]string)
	for k, v := range secretData["values"].(map[string]interface{}) {
		parsedValues[k] = v.(string)
	}
	secretResp.Values = parsedValues

	return secretResp, nil
}

func (ss *SecretStore) List(organizationID string) ([]SecretsItemResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "ListSecret"})
	log.Info("Listing secrets")
	responseItems := make([]SecretsItemResponse, 0)

	log.Debugf("Searching for organizations secrets [%s]", organizationID)
	orgSecretPath := fmt.Sprintf("secret/orgs/%s", organizationID)

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
					ID:         key.(string),
					Name:       secretData["name"].(string),
					SecretType: secretData["type"].(string),
				}
				responseItems = append(responseItems, sir)
			}
		}
	} else {
		return responseItems, nil
	}

	return responseItems, nil
}
