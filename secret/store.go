package secret

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/banzaicloud/bank-vaults/vault"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/constants"
	"github.com/banzaicloud/pipeline/secret/verify"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
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
	Name    string            `json:"name" binding:"required"`
	Type    string            `json:"type" binding:"required"`
	Values  map[string]string `json:"values" binding:"required"`
	Tags    []string          `json:"tags"`
	Version *int              `json:"version,omitempty"`
}

// SecretsItemResponse for GetSecret (no API endpoint for this!)
type SecretsItemResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Values    map[string]string `json:"values"`
	Tags      []string          `json:"tags"`
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"createdAt"`
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

func generateSecretID(request *CreateSecretRequest) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(request.Name)))
}

// RepoTag creates a secret tag for repository mapping
func RepoTag(repo string) string {
	return fmt.Sprint("repo:", repo)
}

// Validate SecretRequest
func (r *CreateSecretRequest) Validate(verifier verify.Verifier) error {
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

	path := secretMetadataPath(organizationID, secretID)

	log.Debugln("Delete secret:", path)

	if _, err := ss.logical.Delete(path); err != nil {
		return errors.Wrap(err, "Error during deleting secret")
	}

	return nil
}

// Save secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Store(organizationID string, value *CreateSecretRequest) (string, error) {
	log := logger.WithFields(logrus.Fields{"tag": "StoreSecret"})

	secretID := generateSecretID(value)
	path := secretDataPath(organizationID, secretID)

	log.Debugln("Storing secret:", path)

	sort.Strings(value.Tags)

	value.Version = nil

	data := vault.NewData(0, map[string]interface{}{"value": value})

	if _, err := ss.logical.Write(path, data); err != nil {
		return "", errors.Wrap(err, "Error during storing secret")
	}

	return secretID, nil
}

// Update secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Update(organizationID, secretID string, value *CreateSecretRequest) error {
	log := logger.WithFields(logrus.Fields{"tag": "UpdateSecret"})

	path := secretDataPath(organizationID, secretID)

	log.Debugln("Update secret:", path)

	sort.Strings(value.Tags)

	version := *value.Version
	value.Version = nil

	data := vault.NewData(version, map[string]interface{}{"value": value})

	if _, err := ss.logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during updating secret")
	}

	return nil
}

func parseSecret(secretID string, secret *vaultapi.Secret, values bool) (*SecretsItemResponse, error) {
	data := cast.ToStringMap(secret.Data["data"])
	metadata := cast.ToStringMap(secret.Data["metadata"])
	value := data["value"].(map[string]interface{})
	sname := value["name"].(string)
	stype := value["type"].(string)
	stags := cast.ToStringSlice(value["tags"])
	version, _ := metadata["version"].(json.Number).Int64()
	createdAt, err := time.Parse(time.RFC3339, metadata["created_time"].(string))
	if err != nil {
		return nil, err
	}

	sir := SecretsItemResponse{
		ID:        secretID,
		Name:      sname,
		Type:      stype,
		Tags:      stags,
		Values:    cast.ToStringMapString(value["values"]),
		Version:   int(version),
		CreatedAt: createdAt,
	}

	if !values {
		// Clear the values otherwise
		for k := range sir.Values {
			sir.Values[k] = "<hidden>"
		}
	}

	return &sir, nil
}

// Retrieve secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Get(organizationID string, secretID string) (*SecretsItemResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "GetSecret"})

	path := secretDataPath(organizationID, secretID)

	log.Debugln("Get secret:", path)

	secret, err := ss.logical.Read(path)

	if err != nil {
		return nil, errors.Wrap(err, "Error during reading secret")
	}

	if secret == nil {
		return nil, fmt.Errorf("there's no secret with this id: %s", secretID)
	}

	return parseSecret(secretID, secret, true)
}

// ListSecretsQuery represent a secret listing filter
type ListSecretsQuery struct {
	Type   string `form:"type"`
	Tag    string `form:"tag"`
	Values bool   `form:"values"`
}

// List secret secret/orgs/:orgid:/ scope
func (ss *secretStore) List(orgid string, query *ListSecretsQuery) ([]*SecretsItemResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "ListSecret"})

	log.Debugf("Searching for secrets [orgid: %s, query: %#v]", orgid, query)

	listPath := fmt.Sprintf("secret/metadata/orgs/%s", orgid)

	responseItems := []*SecretsItemResponse{}

	list, err := ss.logical.List(listPath)
	if err != nil {
		log.Errorf("Error listing secrets: %s", err.Error())
		return nil, err
	}

	if list != nil {

		keys := cast.ToStringSlice(list.Data["keys"])

		for _, secretID := range keys {

			if secret, err := ss.logical.Read(secretDataPath(orgid, secretID)); err != nil {

				log.Errorf("Error listing secrets: %s", err.Error())
				return nil, err

			} else if secret != nil {

				sir, err := parseSecret(secretID, secret, query.Values)
				if err != nil {
					return nil, err
				}

				if (query.Type == constants.AllSecrets || sir.Type == query.Type) &&
					(query.Tag == "" || hasTag(sir.Tags, query.Tag)) &&
					(IsForbiddenTag(sir.Tags) == nil) {

					responseItems = append(responseItems, sir)
				}
			}
		}
	}

	return responseItems, nil
}

func secretDataPath(organizationID, secretID string) string {
	return fmt.Sprintf("secret/data/orgs/%s/%s", organizationID, secretID)
}

func secretMetadataPath(organizationID, secretID string) string {
	return fmt.Sprintf("secret/metadata/orgs/%s/%s", organizationID, secretID)
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

// IsCASError detects if the underlying Vault error is caused by a CAS failure
func IsCASError(err error) bool {
	return strings.HasSuffix(err.Error(), "check-and-set parameter did not match the current version")
}
