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

// Store object that wraps up vault logical store
var Store *secretStore

// Validated secret types
const (
	Amazon     = "AMAZON_SECRET"
	Azure      = "AZURE_SECRET"
	Google     = "GOOGLE_SECRET"
	General    = "GENERAL_SECRET"
	Kubernetes = "KUBERNETES_SECRET"
)

// All supported secret types in a slice to help in validate (in list secrets endpoint)
var AllTypes = []string{
	Amazon,
	Azure,
	Google,
	General,
	Kubernetes,
}

func init() {
	logger = config.Logger()
	Store = newVaultSecretStore()
}

type secretStore struct {
	client  *vault.Client
	logical *vaultapi.Logical
}

// CreateSecretResponse API response for AddSecrets
type CreateSecretResponse struct {
	Name       string `json:"name" binding:"required"`
	SecretType string `json:"type" binding:"required"`
	SecretID   string `json:"secret_id"`
}

// CreateSecretRequest param for Store.Store
type CreateSecretRequest struct {
	Name       string            `json:"name" binding:"required"`
	SecretType string            `json:"type" binding:"required"`
	Values     map[string]string `json:"values" binding:"required"`
}

// ListSecretsResponse for API response for ListSecrets
type ListSecretsResponse struct {
	Secrets []SecretsItemResponse `json:"secrets"`
}

// SecretsItemResponse for GetSecret (no API endpoint for this!)
type SecretsItemResponse struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	SecretType string            `json:"type"`
	Values     map[string]string `json:"-"`
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

// GenerateSecretID uuid for new secrets
func GenerateSecretID() string {
	log := logger.WithFields(logrus.Fields{"tag": "Secret"})
	log.Debug("Generating secret id")
	return uuid.NewV4().String()
}

// DefaultRules key matching for types
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
		"type",
		"project_id",
		"private_key_id",
		"private_key",
		"client_email",
		"client_id",
		"auth_uri",
		"token_uri",
		"auth_provider_x509_cert_url",
		"client_x509_cert_url",
	},
}

// Validate SecretRequest
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

// Delete secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Delete(organizationID, secretID string) error {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteSecret"})
	log.Debugf("Delete sectret: %s", fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID))
	_, err := ss.logical.Delete(fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID))
	return err
}

// Save secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Store(organizationID, secretID string, value CreateSecretRequest) error {
	log := logger.WithFields(logrus.Fields{"tag": "StoreSecret"})
	log.Infof("Storing secret")
	path := fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)
	data := map[string]interface{}{"value": value}
	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during storing secret")
	}
	return nil
}

// Retrieve secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Get(organizationID string, secretID string) (*SecretsItemResponse, error) {
	secretPath := fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)
	secret, err := ss.logical.Read(secretPath)
	if err != nil {
		return nil, err
	}
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

// List secret secret/orgs/:orgid:/ scope
func (ss *secretStore) List(organizationID, secretType string) ([]SecretsItemResponse, error) {
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
				sType := secretData["type"].(string)
				if len(secretType) == 0 || sType == secretType {
					sir := SecretsItemResponse{
						ID:         key.(string),
						Name:       secretData["name"].(string),
						SecretType: sType,
					}
					responseItems = append(responseItems, sir)
				}
			}
		}
	} else {
		return responseItems, nil
	}
	return responseItems, nil
}
