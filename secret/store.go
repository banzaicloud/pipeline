package secret

import (
	"fmt"

	"github.com/banzaicloud/bank-vaults/vault"
	btypes "github.com/banzaicloud/banzai-types/constants"
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
	General = "GENERAL_SECRET"
)

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

// AllowedFilteredSecretTypesResponse for API response for AllowedSecretTypes/:type
type AllowedFilteredSecretTypesResponse struct {
	Keys []string `json:"keys"`
}

// AllowedFilteredSecretTypesResponse for API response for AllowedSecretTypes
type AllowedSecretTypesResponse struct {
	Allowed map[string][]string `json:"allowed"`
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

const repoSecretType = "repo"

// DefaultRules key matching for types
var DefaultRules = map[string][]string{
	btypes.Amazon: {
		AwsAccessKeyId,
		AwsSecretAccessKey,
	},
	btypes.Azure: {
		AzureClientId,
		AzureClientSecret,
		AzureTenantId,
		AzureSubscriptionId,
	},
	btypes.Google: {
		Type,
		ProjectId,
		PrivateKeyId,
		PrivateKey,
		ClientEmail,
		ClientId,
		AuthUri,
		TokenUri,
		AuthX509Url,
		ClientX509Url,
	},
	btypes.Kubernetes: {
		K8SConfig,
	},
	repoSecretType: {
		RepoName,
		RepoSecret,
	},
}

// Amazon keys
const (
	AwsAccessKeyId     = "AWS_ACCESS_KEY_ID"
	AwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

// Azure keys
const (
	AzureClientId       = "AZURE_CLIENT_ID"
	AzureClientSecret   = "AZURE_CLIENT_SECRET"
	AzureTenantId       = "AZURE_TENANT_ID"
	AzureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
)

// Google keys
const (
	Type          = "type"
	ProjectId     = "project_id"
	PrivateKeyId  = "private_key_id"
	PrivateKey    = "private_key"
	ClientEmail   = "client_email"
	ClientId      = "client_id"
	AuthUri       = "auth_uri"
	TokenUri      = "token_uri"
	AuthX509Url   = "auth_provider_x509_cert_url"
	ClientX509Url = "client_x509_cert_url"
)

// Kubernetes keys
const (
	K8SConfig = "K8Sconfig"
)

// Repo keys
const (
	RepoName   = "RepoName"
	RepoSecret = "RepoSecret"
)

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
	log.Debugf("Delete secret: %s", fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID))
	_, err := ss.logical.Delete(fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID))
	return err
}

// Save secret secret/orgs/:orgid:/:id: scope or to secret/orgs/:orgid:/:repo:/:id in case of repo secret
func (ss *secretStore) Store(organizationID, secretID string, value CreateSecretRequest) error {
	log := logger.WithFields(logrus.Fields{"tag": "StoreSecret"})
	log.Infof("Storing secret")
	var path string
	if value.SecretType != repoSecretType {
		path = fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)
	} else {
		path = fmt.Sprintf("secret/orgs/%s/%s/%s", organizationID, value.Values[RepoName], secretID)
		delete(value.Values, RepoName)
	}
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

	if secret == nil {
		return nil, fmt.Errorf("there's no secret with this id: %s", secretID)
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
func (ss *secretStore) List(organizationID, secretType string, reponame string) ([]SecretsItemResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "ListSecret"})
	log.Info("Listing secrets")
	responseItems := make([]SecretsItemResponse, 0)

	log.Debugf("Searching for organizations secrets [%s]", organizationID)
	orgSecretPath := fmt.Sprintf("secret/orgs/%s/%s", organizationID, reponame)

	if secret, err := ss.logical.List(orgSecretPath); err != nil {
		log.Errorf("Error listing secrets: %s", err.Error())
		return nil, err
	} else if secret != nil {
		keys := secret.Data["keys"].([]interface{})
		for _, key := range keys {
			secretID := key.(string)
			if readSecret, err := ss.logical.Read(orgSecretPath + "/" + secretID); err != nil {
				log.Errorf("Error listing secrets: %s", err.Error())
				return nil, err
			} else if readSecret != nil {
				secretData := readSecret.Data["value"].(map[string]interface{})
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

// GetValue returns the value under key
func (s *SecretsItemResponse) GetValue(key string) string {
	return s.Values[key]
}

func (s *SecretsItemResponse) ValidateSecretType(validType string) error {
	if s.SecretType != validType {
		return MissmatchError{
			SecretType: s.SecretType,
			ValidType:  validType,
		}
	}
	return nil
}

type MissmatchError struct {
	Err        error
	SecretType string
	ValidType  string
}

func (m MissmatchError) Error() string {
	if m.Err == nil {
		return fmt.Sprintf("missmatch secret type %s versus %s", m.SecretType, m.ValidType)
	}
	return m.Err.Error()
}
