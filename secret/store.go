// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/tls"
	"github.com/banzaicloud/bank-vaults/pkg/vault"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	rsaKeySize             = 2048
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	PublicKeyBlockType     = "PUBLIC KEY"
)

// Store object that wraps up vault logical store
// nolint: gochecknoglobals
var Store *secretStore

// RestrictedStore object that wraps the main secret store and restricts access to certain items
// nolint: gochecknoglobals
var RestrictedStore *restrictedSecretStore

// ErrSecretNotExists denotes 'Not Found' errors for secrets
// nolint: gochecknoglobals
var ErrSecretNotExists = fmt.Errorf("There's no secret with this ID")

func init() {
	Store = newVaultSecretStore()
	RestrictedStore = &restrictedSecretStore{Store}
}

type secretStore struct {
	Client  *vault.Client
	Logical *vaultapi.Logical
}

// CreateSecretResponse API response for AddSecrets
type CreateSecretResponse struct {
	Name      string    `json:"name" binding:"required"`
	Type      string    `json:"type" binding:"required"`
	ID        string    `json:"id"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	Version   int       `json:"version,omitempty"`
}

// CreateSecretRequest param for Store.Store
// Only fields with `mapstructure` tag are getting written to Vault
type CreateSecretRequest struct {
	Name      string            `json:"name" binding:"required" mapstructure:"name"`
	Type      string            `json:"type" binding:"required" mapstructure:"type"`
	Values    map[string]string `json:"values" binding:"required" mapstructure:"values"`
	Tags      []string          `json:"tags,omitempty" mapstructure:"tags"`
	Version   *int              `json:"version,omitempty" mapstructure:"-"`
	UpdatedBy string            `json:"updatedBy,omitempty" mapstructure:"updatedBy"`
}

func (r *CreateSecretRequest) MarshalJSON() ([]byte, error) {
	type Alias CreateSecretRequest
	modified := Alias(*r)
	modified.Values = map[string]string{}
	for k := range r.Values {
		modified.Values[k] = ""
	}
	return json.Marshal(&modified)
}

// SecretItemResponse for GetSecret
type SecretItemResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name" mapstructure:"name"`
	Type      string            `json:"type" mapstructure:"type"`
	Values    map[string]string `json:"values" mapstructure:"values"`
	Tags      []string          `json:"tags" mapstructure:"tags"`
	Version   int               `json:"version"`
	UpdatedAt time.Time         `json:"updatedAt"`
	UpdatedBy string            `json:"updatedBy,omitempty" mapstructure:"updatedBy"`
}

// K8SSourceMeta returns the meta information how to use this secret if installed to K8S
func (s *SecretItemResponse) K8SSourceMeta() secretTypes.K8SSourceMeta {
	return secretTypes.K8SSourceMeta{
		Name:     s.Name,
		Sourcing: secretTypes.DefaultRules[s.Type].Sourcing,
	}
}

// GetValue returns the value under key
func (s *SecretItemResponse) GetValue(key string) string {
	return s.Values[key]
}

// ValidateSecretType validates the secret type
func (s *SecretItemResponse) ValidateSecretType(validType string) error {
	if string(s.Type) != validType {

		return MismatchError{
			SecretType: s.Type,
			ValidType:  validType,
		}
	}
	return nil
}

// AllowedFilteredSecretTypesResponse for API response for AllowedSecretTypes/:type
type AllowedFilteredSecretTypesResponse struct {
	Keys secretTypes.Meta `json:"meta"`
}

// AllowedSecretTypesResponse for API response for AllowedSecretTypes
type AllowedSecretTypesResponse map[string]secretTypes.Meta

func newVaultSecretStore() *secretStore {
	role := "pipeline"
	client, err := vault.NewClient(role)
	if err != nil {
		panic(err)
	}
	logical := client.Vault().Logical()
	return &secretStore{Client: client, Logical: logical}
}

// GenerateSecretIDFromName generates a "unique by name per organization" id for Secrets
func GenerateSecretIDFromName(name string) string {
	return string(fmt.Sprintf("%x", sha256.Sum256([]byte(name))))
}

// GenerateSecretID generates a "unique by name per organization" id for Secrets
func GenerateSecretID(request *CreateSecretRequest) string {
	return GenerateSecretIDFromName(request.Name)
}

// Validate SecretRequest
func (r *CreateSecretRequest) Validate(verifier verify.Verifier) error {
	fields, ok := secretTypes.DefaultRules[r.Type]

	if !ok {
		return errors.Errorf("wrong secret type: %s", r.Type)
	}

	for _, field := range fields.Fields {
		if _, ok := r.Values[field.Name]; field.Required && !ok {
			return errors.Errorf("missing key: %s", field.Name)
		}
	}

	if verifier != nil {
		return verifier.VerifySecret()
	}

	return nil
}

// ValidateAsNew validates a create secret request as it was a new secret.
func (r *CreateSecretRequest) ValidateAsNew(verifier verify.Verifier) error {
	fields, ok := secretTypes.DefaultRules[r.Type]

	if !ok {
		return errors.Errorf("wrong secret type: %s", r.Type)
	}

	switch r.Type {
	case secretTypes.TLSSecretType:
		if len(r.Values) < 3 { // Assume secret generation
			if _, ok := r.Values[secretTypes.TLSHosts]; !ok {
				return errors.Errorf("missing key: %s", secretTypes.TLSHosts)
			}
		}

		if len(r.Values) >= 3 { // We expect keys for server TLS (at least)
			for _, field := range []string{secretTypes.CACert, secretTypes.ServerKey, secretTypes.ServerCert} {
				if _, ok := r.Values[field]; !ok {
					return errors.Errorf("missing key: %s", field)
				}
			}
		}

		if len(r.Values) > 3 { // We expect keys for mutual TLS
			for _, field := range []string{secretTypes.ClientKey, secretTypes.ClientCert} {
				if _, ok := r.Values[field]; !ok {
					return errors.Errorf("missing key: %s", field)
				}
			}
		}

	default:
		for _, field := range fields.Fields {
			if _, ok := r.Values[field.Name]; field.Required && !ok {
				return errors.Errorf("missing key: %s", field.Name)
			}
		}
	}

	if verifier != nil {
		return verifier.VerifySecret()
	}

	return nil
}

// DeleteByClusterUID Delete secrets by ClusterUID
func (ss *secretStore) DeleteByClusterUID(orgID uint, clusterUID string) error {
	if clusterUID == "" {
		return errors.New("clusterUID is empty")
	}

	log := log.WithFields(logrus.Fields{"organization": orgID, "clusterUID": clusterUID})

	clusterUIDTag := clusterUIDTag(clusterUID)
	secrets, err := Store.List(orgID,
		&secretTypes.ListSecretsQuery{
			Tags: []string{clusterUIDTag},
		})

	if err != nil {
		log.Errorf("Error during list secrets: %s", err.Error())
		return err
	}

	for _, s := range secrets {
		log := log.WithFields(logrus.Fields{"secret": s.ID, "secretName": s.Name})
		err := Store.Delete(orgID, s.ID)
		if err != nil {
			log.Errorf("Error during delete secret: %s", err.Error())
		}
		log.Infoln("Secret Deleted")
	}

	return nil
}

// Delete secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Delete(organizationID uint, secretID string) error {

	path := secretMetadataPath(organizationID, secretID)

	log.Debugln("Delete secret:", path)

	secret, err := ss.Get(organizationID, secretID)
	if err != nil {
		return errors.Wrap(err, "Error during querying secret before deletion")
	}

	if _, err := ss.Logical.Delete(path); err != nil {
		return errors.Wrap(err, "Error during deleting secret")
	}

	// if type is distribution, unmount all pki engines
	if secret.Type == secretTypes.PKESecretType {
		clusterID := getClusterIDFromTags(secret.Tags)
		basePath := clusterPKIPath(organizationID, clusterID)

		path = fmt.Sprintf("%s/ca", basePath)
		err = ss.Client.Vault().Sys().Unmount(path)
		if err != nil {
			log.Warnf("failed to unmount %s: %s", path, err)
		}

		path = fmt.Sprintf("%s/%s", basePath, secretTypes.KubernetesCACommonName)
		err = ss.Client.Vault().Sys().Unmount(path)
		if err != nil {
			log.Warnf("failed to unmount %s: %s", path, err)
		}

		path = fmt.Sprintf("%s/%s", basePath, secretTypes.EtcdCACommonName)
		err = ss.Client.Vault().Sys().Unmount(path)
		if err != nil {
			log.Warnf("failed to unmount %s: %s", path, err)
		}

		path = fmt.Sprintf("%s/%s", basePath, secretTypes.KubernetesFrontProxyCACommonName)
		err = ss.Client.Vault().Sys().Unmount(path)
		if err != nil {
			log.Warnf("failed to unmount %s: %s", path, err)
		}
	}

	return nil
}

// Save secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Store(organizationID uint, request *CreateSecretRequest) (string, error) {

	// We allow only Kubernetes compatible Secret names
	if errorList := validation.IsDNS1123Subdomain(request.Name); errorList != nil {
		return "", errors.New(errorList[0])
	}

	secretID := GenerateSecretID(request)
	path := secretDataPath(organizationID, secretID)

	if err := ss.generateValuesIfNeeded(organizationID, request); err != nil {
		return "", err
	}

	sort.Strings(request.Tags)

	data, err := secretData(0, request)
	if err != nil {
		return "", err
	}

	if _, err := ss.Logical.Write(path, data); err != nil {
		return "", errors.Wrap(err, "Error during storing secret")
	}

	return secretID, nil
}

// Update secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Update(organizationID uint, secretID string, request *CreateSecretRequest) error {

	if GenerateSecretID(request) != secretID {
		return errors.New("Secret name cannot be changed")
	}

	path := secretDataPath(organizationID, secretID)

	log.Debugln("Update secret:", path)

	sort.Strings(request.Tags)

	// If secret doesn't exists, create it.
	version := 0
	if request.Version != nil {
		version = *request.Version
	}

	data, err := secretData(version, request)
	if err != nil {
		return err
	}

	if _, err := ss.Logical.Write(path, data); err != nil {
		return errors.Wrap(err, "Error during updating secret")
	}

	return nil
}

// GetOrCreate create new secret or get if it's exist. secret/orgs/:orgid:/:id: scope
func (ss *secretStore) GetOrCreate(organizationID uint, value *CreateSecretRequest) (string, error) {
	secretID := GenerateSecretID(value)

	// Try to get the secret version first
	if secret, err := Store.Get(organizationID, secretID); err != nil && err != ErrSecretNotExists {
		log.Errorf("Error during checking secret: %s", err.Error())
		return "", err
	} else if secret != nil {
		return secret.ID, nil
	} else {
		secretID, err = Store.Store(organizationID, value)
		if err != nil {
			log.Errorf("Error during storing secret: %s", err.Error())
			return "", err
		}
	}
	return secretID, nil
}

// CreateOrUpdate create new secret or update if it's exist. secret/orgs/:orgid:/:id: scope
func (ss *secretStore) CreateOrUpdate(organizationID uint, value *CreateSecretRequest) (string, error) {

	secretID := GenerateSecretID(value)

	// Try to get the secret version first
	if secret, err := Store.Get(organizationID, secretID); err != nil && err != ErrSecretNotExists {
		log.Errorf("Error during checking secret: %s", err.Error())
		return "", err
	} else if secret != nil {
		value.Version = &(secret.Version)
		err := Store.Update(organizationID, secretID, value)
		if err != nil {
			log.Errorf("Error during updating secret: %s", err.Error())
			return "", err
		}
	} else {
		secretID, err = Store.Store(organizationID, value)
		if err != nil {
			log.Errorf("Error during storing secret: %s", err.Error())
			return "", err
		}
	}
	return secretID, nil
}

func parseSecret(secretID string, secret *vaultapi.Secret, values bool) (*SecretItemResponse, error) {

	data := cast.ToStringMap(secret.Data["data"])
	metadata := cast.ToStringMap(secret.Data["metadata"])

	version, _ := metadata["version"].(json.Number).Int64()

	updatedAt, err := time.Parse(time.RFC3339, metadata["created_time"].(string))
	if err != nil {
		return nil, err
	}

	response := SecretItemResponse{
		ID:        secretID,
		Version:   int(version),
		UpdatedAt: updatedAt,
		Tags:      []string{},
	}

	if err := mapstructure.Decode(data["value"], &response); err != nil {
		return nil, err
	}

	if !values {
		// Clear the values otherwise
		for k := range response.Values {
			response.Values[k] = "<hidden>"
		}
	}

	return &response, nil
}

// Retrieve secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Get(organizationID uint, secretID string) (*SecretItemResponse, error) {

	path := secretDataPath(organizationID, secretID)

	secret, err := ss.Logical.Read(path)

	if err != nil {
		return nil, errors.Wrap(err, "Error during reading secret")
	}

	if secret == nil {
		return nil, ErrSecretNotExists
	}

	return parseSecret(secretID, secret, true)
}

// Retrieve secret by secret Name secret/orgs/:orgid:/:id: scope
func (ss *secretStore) GetByName(organizationID uint, name string) (*SecretItemResponse, error) {

	secretID := GenerateSecretIDFromName(name)
	secret, err := Store.Get(organizationID, secretID)
	if err == ErrSecretNotExists {
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "Error during reading secret")
	}

	if secret == nil {
		return nil, ErrSecretNotExists
	}

	return secret, nil
}

func (ss *secretStore) getSecretIDs(orgid uint, query *secretTypes.ListSecretsQuery) ([]string, error) {
	if len(query.IDs) > 0 {
		return query.IDs, nil
	}

	listPath := fmt.Sprintf("secret/metadata/orgs/%d", orgid)

	list, err := ss.Logical.List(listPath)
	if err != nil {
		return nil, err
	}

	if list != nil {
		keys := cast.ToStringSlice(list.Data["keys"])
		res := make([]string, len(keys))
		for i, key := range keys {
			res[i] = string(key)
		}
		return res, nil
	}

	return nil, nil
}

// List secret secret/orgs/:orgid:/ scope
func (ss *secretStore) List(orgid uint, query *secretTypes.ListSecretsQuery) ([]*SecretItemResponse, error) {

	log.Debugf("Searching for secrets [orgid: %d, query: %#v]", orgid, query)

	secretIDs, err := ss.getSecretIDs(orgid, query)
	if err != nil {
		log.Errorf("Error listing secrets: %s", err.Error())
		return nil, err
	}

	responseItems := []*SecretItemResponse{}

	for _, secretID := range secretIDs {

		if secret, err := ss.Logical.Read(secretDataPath(orgid, secretID)); err != nil {

			log.Errorf("Error listing secrets: %s", err.Error())
			return nil, err

		} else if secret != nil {

			sir, err := parseSecret(secretID, secret, query.Values)
			if err != nil {
				return nil, err
			}

			if (query.Type == secretTypes.AllSecrets || sir.Type == query.Type) && hasTags(sir.Tags, query.Tags) {
				responseItems = append(responseItems, sir)
			}
		}
	}

	return responseItems, nil
}

func secretData(version int, request *CreateSecretRequest) (map[string]interface{}, error) {
	valueData := map[string]interface{}{}

	if err := mapstructure.Decode(request, &valueData); err != nil {
		return nil, errors.Wrap(err, "Error during encoding secret")
	}

	return vault.NewData(version, map[string]interface{}{"value": valueData}), nil
}

func secretDataPath(organizationID uint, secretID string) string {
	return fmt.Sprintf("secret/data/orgs/%d/%s", organizationID, secretID)
}

func secretMetadataPath(organizationID uint, secretID string) string {
	return fmt.Sprintf("secret/metadata/orgs/%d/%s", organizationID, secretID)
}

func hasTags(tags []string, searchingTag []string) bool {
	var isOK bool
	for _, t := range searchingTag {
		index := sort.SearchStrings(tags, t)
		isOK = index < len(tags) && tags[index] == t
		if !isOK {
			return false
		}
	}

	return true
}

// MismatchError describe a secret error where the given and expected secret type is not equal
type MismatchError struct {
	Err        error
	SecretType string
	ValidType  string
}

func (m MismatchError) Error() string {
	if m.Err == nil {
		return fmt.Sprintf("missmatch secret type %s versus %s", m.SecretType, m.ValidType)
	}
	return m.Err.Error()
}

// IsCASError detects if the underlying Vault error is caused by a CAS failure
func IsCASError(err error) bool {
	return strings.Contains(err.Error(), "check-and-set parameter did not match the current version")
}

func (ss *secretStore) generateValuesIfNeeded(organizationID uint, value *CreateSecretRequest) error {
	if value.Type == secretTypes.TLSSecretType && len(value.Values) <= 2 {
		// If we are not storing a full TLS secret instead of it's a request to generate one

		validity := value.Values[secretTypes.TLSValidity]
		if validity == "" {
			validity = viper.GetString("tls.validity")
		}

		cc, err := tls.GenerateTLS(value.Values[secretTypes.TLSHosts], validity)
		if err != nil {
			return errors.Wrap(err, "Error during generating TLS secret")
		}

		err = mapstructure.Decode(cc, &value.Values)
		if err != nil {
			return errors.Wrap(err, "Error during decoding TLS secret")
		}

	} else if value.Type == secretTypes.PasswordSecretType {
		// Generate a password if needed (if password is in method,length)

		if value.Values[secretTypes.Password] == "" {
			value.Values[secretTypes.Password] = DefaultPasswordFormat
		}

		methodAndLength := strings.Split(value.Values[secretTypes.Password], ",")
		if len(methodAndLength) == 2 {
			length, err := strconv.Atoi(methodAndLength[1])
			if err != nil {
				return err
			}
			password, err := RandomString(methodAndLength[0], length)
			if err != nil {
				return err
			}
			value.Values[secretTypes.Password] = password
		}

	} else if value.Type == secretTypes.HtpasswdSecretType {
		// Generate a password if needed otherwise store the htaccess file if provided

		if _, ok := value.Values[secretTypes.HtpasswdFile]; !ok {

			username := value.Values[secretTypes.Username]
			if value.Values[secretTypes.Password] == "" {
				password, err := RandomString("randAlphaNum", 12)
				if err != nil {
					return err
				}
				value.Values[secretTypes.Password] = password
			}

			passwordHash, err := bcrypt.GenerateFromPassword([]byte(value.Values[secretTypes.Password]), bcrypt.DefaultCost)
			if err != nil {
				return err
			}

			value.Values[secretTypes.HtpasswdFile] = fmt.Sprintf("%s:%s", username, string(passwordHash))
		}

	} else if value.Type == secretTypes.PKESecretType {
		clusterID := getClusterIDFromTags(value.Tags)
		if clusterID == "" {
			return errors.New("clusterID is missing from the tags")
		}

		mountInput := vaultapi.MountInput{
			Type:        "pki",
			Description: fmt.Sprintf("root PKI engine for cluster %s", clusterID),
			Config: vaultapi.MountConfigInput{
				MaxLeaseTTL:     "43801h",
				DefaultLeaseTTL: "43801h",
			},
		}

		// Mount a separate PKI engine for the cluster
		basePath := clusterPKIPath(organizationID, clusterID)
		path := fmt.Sprintf("%s/ca", basePath)

		err := ss.Client.Vault().Sys().Mount(path, &mountInput)
		if err != nil {
			return errors.Wrapf(err, "Error mounting pki engine for cluster %s", clusterID)
		}

		// Generate the root CA
		rootCAData := map[string]interface{}{
			"common_name": fmt.Sprintf("cluster-%s-ca", clusterID),
		}

		_, err = ss.Logical.Write(fmt.Sprintf("%s/root/generate/internal", path), rootCAData)
		if err != nil {
			// Unmount the pki engine first
			if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
				log.Warnf("failed to unmount %s: %s", path, err)
			}
			return errors.Wrapf(err, "Error generating root CA for cluster %s", clusterID)
		}

		// Get root CA
		rootCA, err := ss.Logical.Read(fmt.Sprintf("%s/cert/ca", path))
		if err != nil {
			// Unmount the pki engine first
			if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
				log.Warnf("failed to unmount %s: %s", path, err)
			}
			return errors.Wrapf(err, "Error reading root CA for cluster %s", clusterID)
		}
		ca := rootCA.Data["certificate"].(string)

		// Generate the intermediate CAs
		kubernetesCA, err := ss.generateIntermediateCert(clusterID, basePath, secretTypes.KubernetesCACommonName)
		if err != nil {
			// Unmount the pki backend first
			if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
				log.Warnf("failed to unmount %s: %s", path, err)
			}
			return err
		}

		etcdCA, err := ss.generateIntermediateCert(clusterID, basePath, secretTypes.EtcdCACommonName)
		if err != nil {
			// Unmount the pki backend first
			if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
				log.Warnf("failed to unmount %s: %s", path, err)
			}
			return err
		}

		frontProxyCA, err := ss.generateIntermediateCert(clusterID, basePath, secretTypes.KubernetesFrontProxyCACommonName)
		if err != nil {
			// Unmount the pki backend first
			if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
				log.Warnf("failed to unmount %s: %s", path, err)
			}
			return err
		}

		saPub, saPriv, err := generateSAKeyPair(clusterID)
		if err != nil {
			return err
		}

		value.Values[secretTypes.KubernetesCAKey] = kubernetesCA.Key
		value.Values[secretTypes.KubernetesCACert] = kubernetesCA.Cert + "\n" + ca
		value.Values[secretTypes.KubernetesCASigningCert] = kubernetesCA.Cert
		value.Values[secretTypes.EtcdCAKey] = etcdCA.Key
		value.Values[secretTypes.EtcdCACert] = etcdCA.Cert + "\n" + ca
		value.Values[secretTypes.FrontProxyCAKey] = frontProxyCA.Key
		value.Values[secretTypes.FrontProxyCACert] = frontProxyCA.Cert + "\n" + ca
		value.Values[secretTypes.SAPub] = saPub
		value.Values[secretTypes.SAKey] = saPriv
	}

	return nil
}

func (ss *secretStore) generateIntermediateCert(clusterID, basePath, commonName string) (*certificate, error) {
	mountInput := vaultapi.MountInput{
		Type:        "pki",
		Description: fmt.Sprintf("%s intermediate PKI engine for cluster %s", commonName, clusterID),
		Config: vaultapi.MountConfigInput{
			MaxLeaseTTL:     "43800h",
			DefaultLeaseTTL: "43800h",
		},
	}

	path := fmt.Sprintf("%s/%s", basePath, commonName)

	// Each intermediate and ca cert needs it's own pki mount, see:
	// https://github.com/hashicorp/vault/issues/1586#issuecomment-230300216
	err := ss.Client.Vault().Sys().Mount(path, &mountInput)
	if err != nil {
		return nil, errors.Wrapf(err, "error mounting %s intermediate pki engine for cluster %s", commonName, clusterID)
	}

	caData := map[string]interface{}{
		"common_name": commonName,
	}

	caSecret, err := ss.Logical.Write(fmt.Sprintf("%s/intermediate/generate/exported", path), caData)
	if err != nil {
		// Unmount the pki backend first
		if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
			log.Warnf("failed to unmount %s: %s", path, err)
		}
		return nil, errors.Wrapf(err, "error generating %s intermediate cert for cluster %s", commonName, clusterID)
	}

	caSignData := map[string]interface{}{
		"csr":    caSecret.Data["csr"],
		"format": "pem_bundle",
	}

	caCertSecret, err := ss.Logical.Write(fmt.Sprintf("%s/ca/root/sign-intermediate", basePath), caSignData)
	if err != nil {
		// Unmount the pki backend first
		if err := ss.Client.Vault().Sys().Unmount(path); err != nil {
			log.Warnf("failed to unmount %s: %s", path, err)
		}
		return nil, errors.Wrapf(err, "error signing %s intermediate cert for cluster %s", commonName, clusterID)
	}

	return &certificate{
		Key:  caSecret.Data["private_key"].(string),
		Cert: caCertSecret.Data["certificate"].(string),
	}, nil
}

type certificate struct {
	Cert string
	Key  string
}

const clusterIDTagName = "clusterID"
const clusterUIDTagName = "clusterUID"

func clusterUIDTag(clusterUID string) string {
	return fmt.Sprintf("%s:%s", clusterUIDTagName, clusterUID)
}

func getClusterIDFromTags(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, clusterIDTagName+":") {
			return strings.TrimPrefix(tag, clusterIDTagName+":")
		}
	}

	// This should never happen
	return ""
}

func clusterPKIPath(organizationID uint, clusterID string) string {
	return fmt.Sprintf("clusters/%d/%s/pki", organizationID, clusterID)
}

func generateSAKeyPair(clusterID string) (pub, priv string, err error) {
	pk, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return "", "", errors.Wrapf(err, "Error generating SA key pair for cluster %s, generate key failed", clusterID)
	}

	saPub, err := encodePublicKeyPEM(&pk.PublicKey)
	if err != nil {
		return "", "", errors.Wrapf(err, "Error generating SA key pair for cluster %s, encode public key failed", clusterID)
	}
	saPriv := encodePrivateKeyPEM(pk)

	return string(saPub), string(saPriv), nil
}

func encodePublicKeyPEM(key *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:  PublicKeyBlockType,
		Bytes: der,
	}
	return pem.EncodeToMemory(&block), nil
}

func encodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  RSAPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}
