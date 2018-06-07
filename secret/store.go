package secret

import (
	"fmt"
	"sort"

	"github.com/banzaicloud/bank-vaults/vault"
	"github.com/banzaicloud/pipeline/auth/cloud"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/constants"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

var logger *logrus.Logger

// Store object that wraps up vault logical store
var Store *secretStore

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
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
	ID   string `json:"id"`
}

// CreateSecretRequest param for Store.Store
type CreateSecretRequest struct {
	Name   string            `json:"name" binding:"required"`
	Type   string            `json:"type" binding:"required"`
	Values map[string]string `json:"values" binding:"required"`
	Tags   []string          `json:"tags"`
}

// ListSecretsResponse for API response for ListSecrets
type ListSecretsResponse struct {
	Secrets []SecretsItemResponse `json:"secrets"`
}

// SecretsItemResponse for GetSecret (no API endpoint for this!)
type SecretsItemResponse struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Values map[string]string `json:"values"`
	Tags   []string          `json:"tags"`
}

// AllowedFilteredSecretTypesResponse for API response for AllowedSecretTypes/:type
type AllowedFilteredSecretTypesResponse struct {
	Keys []string `json:"keys"`
}

// AllowedSecretTypesResponse for API response for AllowedSecretTypes
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
	return uuid.NewV4().String()
}

// RepoTag creates a secret tag for repository mapping
func RepoTag(repo string) string {
	return fmt.Sprint("repo:", repo)
}

// Validate SecretRequest
func (r *CreateSecretRequest) Validate(verifier cloud.Verifier) error {
	requiredKeys, ok := constants.DefaultRules[r.Type]

	if !ok {
		return errors.Errorf("wrong secret type: %s", r.Type)
	}

	for _, key := range requiredKeys {
		if _, ok := r.Values[key]; !ok {
			return errors.Errorf("missing key: %s", key)
		}
	}

	if verifier != nil {
		return verifier.VerifySecret()
	}

	return nil
}

// Delete secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Delete(organizationID, secretID string) error {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteSecret"})

	path := secretPath(organizationID, secretID)

	log.Debugln("Delete secret:", path)

	if _, err := ss.logical.Delete(path); err != nil {
		return errors.Wrap(err, "Error during deleting secret")
	}

	return nil
}

// Save secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Store(organizationID, secretID string, value *CreateSecretRequest) error {
	log := logger.WithFields(logrus.Fields{"tag": "StoreSecret"})

	path := secretPath(organizationID, secretID)

	log.Debugln("Storing secret:", path)

	sort.Strings(value.Tags)

	data := map[string]interface{}{"value": value}

	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during storing secret")
	}

	return nil
}

// Update secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Update(organizationID, secretID string, value *CreateSecretRequest) error {
	log := logger.WithFields(logrus.Fields{"tag": "UpdateSecret"})

	path := secretPath(organizationID, secretID)

	log.Debugln("Update secret:", path)

	sort.Strings(value.Tags)

	data := map[string]interface{}{"value": *value}

	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during updating secret")
	}

	return nil
}

// Retrieve secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Get(organizationID string, secretID string) (*SecretsItemResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "GetSecret"})

	path := secretPath(organizationID, secretID)

	log.Debugln("Get secret:", path)

	secret, err := ss.logical.Read(path)

	if err != nil {
		return nil, errors.Wrap(err, "Error during reading secret")
	}

	if secret == nil {
		return nil, fmt.Errorf("there's no secret with this id: %s", secretID)
	}

	data := secret.Data["value"].(map[string]interface{})
	secretResp := &SecretsItemResponse{
		ID:   secretID,
		Name: data["name"].(string),
		Type: data["type"].(string),
		Tags: cast.ToStringSlice(data["tags"]),
	}

	secretResp.Values = cast.ToStringMapString(data["values"])

	return secretResp, nil
}

// ListSecretsQuery represent a secret listing filter
type ListSecretsQuery struct {
	Type   string `form:"type"`
	Tag    string `form:"tag"`
	Values bool   `form:"values"`
}

// List secret secret/orgs/:orgid:/ scope
func (ss *secretStore) List(organizationID string, query *ListSecretsQuery) ([]SecretsItemResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "ListSecret"})

	log.Debugf("Searching for secrets [orgid: %s, query: %#v]", organizationID, query)

	path := fmt.Sprintf("secret/orgs/%s", organizationID)

	var responseItems []SecretsItemResponse

	list, err := ss.logical.List(path)
	if err != nil {
		log.Errorf("Error listing secrets: %s", err.Error())
		return nil, err
	}

	if list != nil {

		keys := cast.ToStringSlice(list.Data["keys"])

		for _, secretID := range keys {

			if secret, err := ss.logical.Read(path + "/" + secretID); err != nil {

				log.Errorf("Error listing secrets: %s", err.Error())
				return nil, err

			} else if secret != nil {

				data := secret.Data["value"].(map[string]interface{})
				sname := data["name"].(string)
				stype := data["type"].(string)
				stags := cast.ToStringSlice(data["tags"])

				if (query.Type == constants.AllSecrets || stype == query.Type) && (query.Tag == "" || hasTag(stags, query.Tag)) {

					sir := SecretsItemResponse{
						ID:     secretID,
						Name:   sname,
						Type:   stype,
						Tags:   stags,
						Values: cast.ToStringMapString(data["values"]),
					}

					if !query.Values {
						// Clear the values otherwise
						for k := range sir.Values {
							sir.Values[k] = "<hidden>"
						}
					}

					err := IsForbiddenTag(stags)
					if err != nil {
						log.Debugf("Secret[%s] with forbidden tag(s). Do not add to list", secretID)
					} else {
						responseItems = append(responseItems, sir)
					}

				}
			}
		}
	}

	return responseItems, nil
}

func secretPath(organizationID, secretID string) string {
	return fmt.Sprintf("secret/orgs/%s/%s", organizationID, secretID)
}

func hasTag(tags []string, tag string) bool {
	index := sort.SearchStrings(tags, tag)
	return index < len(tags) && tags[index] == tag
}

// GetValue returns the value under key
func (s *SecretsItemResponse) GetValue(key string) string {
	return s.Values[key]
}

// ValidateSecretType validates the secret type
func (s *SecretsItemResponse) ValidateSecretType(validType string) error {
	if string(s.Type) != validType {
		return MissmatchError{
			SecretType: s.Type,
			ValidType:  validType,
		}
	}
	return nil
}

// MissmatchError describe a secret error where the given and expected secret type is not equal
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

// ForbiddenError describes a secret error where it contains forbidden tag
type ForbiddenError struct {
	ForbiddenTag string
}

func (f ForbiddenError) Error() string {
	return fmt.Sprintf("secret contains a forbidden tag: %s", f.ForbiddenTag)
}

// IsForbiddenTag is looking for forbidden tags
func IsForbiddenTag(tags []string) error {
	for _, tag := range tags {
		for _, forbiddenTag := range constants.ForbiddenTags {
			if tag == forbiddenTag {
				return ForbiddenError{
					ForbiddenTag: tag,
				}
			}
		}
	}
	return nil
}
