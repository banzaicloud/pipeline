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
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/tls"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

// Store object that wraps up vault logical store
// nolint: gochecknoglobals
var Store *secretStore

// ErrSecretNotExists denotes 'Not Found' errors for secrets
// nolint: gochecknoglobals
var ErrSecretNotExists = fmt.Errorf("There's no secret with this ID")

// InitSecretStore initializes the global secret store.
func InitSecretStore(store secret.Store, pkeSecreter PkeSecreter) {
	Store = &secretStore{
		SecretStore: store,
		PkeSecreter: pkeSecreter,
	}
}

// PkeSecreter is a temporary interface for splitting the PKE secret generation/deletion code from the legacy secret store.
type PkeSecreter interface {
	GeneratePkeSecret(organizationID uint, tags []string) (map[string]string, error)
	DeletePkeSecret(organizationID uint, tags []string) error
}

type secretStore struct {
	SecretStore secret.Store
	PkeSecreter PkeSecreter
}

// CreateSecretRequest param for secretStore.Store
// Only fields with `mapstructure` tag are getting written to Vault
type CreateSecretRequest struct {
	Name      string            `json:"name" binding:"required" mapstructure:"name"`
	Type      string            `json:"type" binding:"required" mapstructure:"type"`
	Values    map[string]string `json:"values" binding:"required" mapstructure:"values"`
	Tags      []string          `json:"tags,omitempty" mapstructure:"tags"`
	Version   int               `json:"version,omitempty" mapstructure:"-"`
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

// ValidateSecretType validates the secret type
func ValidateSecretType(s *SecretItemResponse, validType string) error {
	if string(s.Type) != validType {

		return MismatchError{
			SecretType: s.Type,
			ValidType:  validType,
		}
	}
	return nil
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
	fields, ok := secrettype.DefaultRules[r.Type]

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
	fields, ok := secrettype.DefaultRules[r.Type]

	if !ok {
		return errors.Errorf("wrong secret type: %s", r.Type)
	}

	switch r.Type {
	case secrettype.TLSSecretType:
		if len(r.Values) < 3 { // Assume secret generation
			if _, ok := r.Values[secrettype.TLSHosts]; !ok {
				return errors.Errorf("missing key: %s", secrettype.TLSHosts)
			}
		}

		if len(r.Values) >= 3 { // We expect keys for server TLS (at least)
			for _, field := range []string{secrettype.CACert, secrettype.ServerKey, secrettype.ServerCert} {
				if _, ok := r.Values[field]; !ok {
					return errors.Errorf("missing key: %s", field)
				}
			}
		}

		if len(r.Values) > 3 { // We expect keys for mutual TLS
			for _, field := range []string{secrettype.ClientKey, secrettype.ClientCert} {
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
	secrets, err := ss.List(orgID,
		&ListSecretsQuery{
			Tags: []string{clusterUIDTag},
		})

	if err != nil {
		log.Errorf("Error during list secrets: %s", err.Error())
		return err
	}

	for _, s := range secrets {
		log := log.WithFields(logrus.Fields{"secret": s.ID, "secretName": s.Name})
		err := ss.Delete(orgID, s.ID)
		if err != nil {
			log.Errorf("Error during delete secret: %s", err.Error())
		}
		log.Infoln("Secret Deleted")
	}

	return nil
}

// Delete secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Delete(organizationID uint, secretID string) error {
	log.WithFields(logrus.Fields{
		"organizationId": organizationID,
		"secretId":       secretID,
	}).Debugln("deleting secret")

	secret, err := ss.Get(organizationID, secretID)
	if err == ErrSecretNotExists { // Already deleted
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "Error during querying secret before deletion")
	}

	if err := ss.SecretStore.Delete(context.Background(), organizationID, secretID); err != nil {
		return err
	}

	// if type is distribution, unmount all pki engines
	if secret.Type == secrettype.PKESecretType {
		err := ss.PkeSecreter.DeletePkeSecret(organizationID, secret.Tags)
		if err != nil {
			return err
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

	if err := ss.generateValuesIfNeeded(organizationID, request); err != nil {
		return "", err
	}

	model := secret.Model{
		ID:        secretID,
		Name:      request.Name,
		Type:      request.Type,
		Values:    request.Values,
		Tags:      request.Tags,
		UpdatedBy: request.UpdatedBy,
	}

	if err := ss.SecretStore.Create(context.Background(), organizationID, model); err != nil {
		return "", err
	}

	return secretID, nil
}

// Update secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Update(organizationID uint, secretID string, request *CreateSecretRequest) error {
	if GenerateSecretID(request) != secretID {
		return errors.New("Secret name cannot be changed")
	}

	log.WithFields(logrus.Fields{
		"organizationId": organizationID,
		"secretId":       secretID,
	}).Debugln("updating secret")

	model := secret.Model{
		ID:        secretID,
		Name:      request.Name,
		Type:      request.Type,
		Values:    request.Values,
		Tags:      request.Tags,
		UpdatedBy: request.UpdatedBy,
	}

	if err := ss.SecretStore.Put(context.Background(), organizationID, model); err != nil {
		return err
	}

	return nil
}

// GetOrCreate create new secret or get if it's exist. secret/orgs/:orgid:/:id: scope
func (ss *secretStore) GetOrCreate(organizationID uint, value *CreateSecretRequest) (string, error) {
	secretID := GenerateSecretID(value)

	// Try to get the secret version first
	if secret, err := ss.Get(organizationID, secretID); err != nil && err != ErrSecretNotExists {
		log.Errorf("Error during checking secret: %s", err.Error())
		return "", err
	} else if secret != nil {
		return secret.ID, nil
	} else {
		secretID, err = ss.Store(organizationID, value)
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
	if secret, err := ss.Get(organizationID, secretID); err != nil && err != ErrSecretNotExists {
		log.Errorf("Error during checking secret: %s", err.Error())
		return "", err
	} else if secret != nil {
		value.Version = secret.Version
		err := ss.Update(organizationID, secretID, value)
		if err != nil {
			log.Errorf("Error during updating secret: %s", err.Error())
			return "", err
		}
	} else {
		secretID, err = ss.Store(organizationID, value)
		if err != nil {
			log.Errorf("Error during storing secret: %s", err.Error())
			return "", err
		}
	}
	return secretID, nil
}

// Retrieve secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Get(organizationID uint, secretID string) (*SecretItemResponse, error) {
	model, err := ss.SecretStore.Get(context.Background(), organizationID, secretID)
	if err != nil && errors.As(err, &secret.NotFoundError{}) {
		return nil, ErrSecretNotExists
	} else if err != nil {
		return nil, err
	}

	return &SecretItemResponse{
		ID:        model.ID,
		Name:      model.Name,
		Type:      model.Type,
		Values:    model.Values,
		Tags:      model.Tags,
		Version:   0,
		UpdatedAt: model.UpdatedAt,
		UpdatedBy: model.UpdatedBy,
	}, nil
}

// Retrieve secret by secret Name secret/orgs/:orgid:/:id: scope
func (ss *secretStore) GetByName(organizationID uint, name string) (*SecretItemResponse, error) {
	secretID := GenerateSecretIDFromName(name)

	secret, err := ss.Get(organizationID, secretID)
	if err != nil {
		return nil, ErrSecretNotExists
	}

	return secret, nil
}

// List secret secret/orgs/:orgid:/ scope
func (ss *secretStore) List(orgid uint, query *ListSecretsQuery) ([]*SecretItemResponse, error) {
	log.Debugf("Searching for secrets [orgid: %d, query: %#v]", orgid, query)

	var models []secret.Model
	var err error

	if len(query.IDs) > 0 {
		models = make([]secret.Model, 0, len(query.IDs))

		for _, id := range query.IDs {
			model, err := ss.SecretStore.Get(context.Background(), orgid, id)
			if err != nil {
				return nil, err
			}

			models = append(models, model)
		}
	} else {
		models, err = ss.SecretStore.List(context.Background(), orgid)
		if err != nil {
			log.Errorf("Error listing secrets: %s", err.Error())

			return nil, err
		}
	}

	responseItems := []*SecretItemResponse{}

	for _, model := range models {
		if !((query.Type == secrettype.AllSecrets || model.Type == query.Type) && hasTags(model.Tags, query.Tags)) {
			continue
		}

		responseItem := &SecretItemResponse{
			ID:        model.ID,
			Name:      model.Name,
			Type:      model.Type,
			Values:    model.Values,
			Tags:      model.Tags,
			Version:   0,
			UpdatedAt: model.UpdatedAt,
			UpdatedBy: model.UpdatedBy,
		}

		if !query.Values {
			// Clear the values otherwise
			for k := range responseItem.Values {
				responseItem.Values[k] = "<hidden>"
			}
		}

		responseItems = append(responseItems, responseItem)
	}

	return responseItems, nil
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

func (MismatchError) ServiceError() bool {
	return true
}

// IsCASError detects if the underlying Vault error is caused by a CAS failure
func IsCASError(err error) bool {
	return strings.Contains(err.Error(), "check-and-set parameter did not match the current version") || errors.As(err, &secret.AlreadyExistsError{})
}

func isTLSSecretGenerationNeeded(cr *CreateSecretRequest) bool {
	for k, v := range cr.Values {
		if k != secrettype.TLSHosts && k != secrettype.TLSValidity && v != "" {
			return false
		}
	}
	return true
}

func (ss *secretStore) generateValuesIfNeeded(organizationID uint, value *CreateSecretRequest) error {
	if value.Type == secrettype.TLSSecretType && isTLSSecretGenerationNeeded(value) {
		// If we are not storing a full TLS secret instead of it's a request to generate one

		validity := value.Values[secrettype.TLSValidity]
		if validity == "" {
			validity = global.Config.Secret.TLS.DefaultValidity.String()
		}

		cc, err := tls.GenerateTLS(value.Values[secrettype.TLSHosts], validity)
		if err != nil {
			return errors.Wrap(err, "Error during generating TLS secret")
		}

		err = mapstructure.Decode(cc, &value.Values)
		if err != nil {
			return errors.Wrap(err, "Error during decoding TLS secret")
		}

	} else if value.Type == secrettype.PasswordSecretType {
		// Generate a password if needed (if password is in method,length)

		if value.Values[secrettype.Password] == "" {
			value.Values[secrettype.Password] = DefaultPasswordFormat
		}

		methodAndLength := strings.Split(value.Values[secrettype.Password], ",")
		if len(methodAndLength) == 2 {
			length, err := strconv.Atoi(methodAndLength[1])
			if err != nil {
				return err
			}
			password, err := RandomString(methodAndLength[0], length)
			if err != nil {
				return err
			}
			value.Values[secrettype.Password] = password
		}

	} else if value.Type == secrettype.HtpasswdSecretType {
		// Generate a password if needed otherwise store the htaccess file if provided

		if _, ok := value.Values[secrettype.HtpasswdFile]; !ok {

			username := value.Values[secrettype.Username]
			if value.Values[secrettype.Password] == "" {
				password, err := RandomString("randAlphaNum", 12)
				if err != nil {
					return err
				}
				value.Values[secrettype.Password] = password
			}

			passwordHash, err := bcrypt.GenerateFromPassword([]byte(value.Values[secrettype.Password]), bcrypt.DefaultCost)
			if err != nil {
				return err
			}

			value.Values[secrettype.HtpasswdFile] = fmt.Sprintf("%s:%s", username, string(passwordHash))
		}

	} else if value.Type == secrettype.PKESecretType {
		values, err := ss.PkeSecreter.GeneratePkeSecret(organizationID, value.Tags)
		if err != nil {
			return err
		}

		value.Values = values
	}

	return nil
}

const clusterUIDTagName = "clusterUID"

func clusterUIDTag(clusterUID string) string {
	return fmt.Sprintf("%s:%s", clusterUIDTagName, clusterUID)
}
