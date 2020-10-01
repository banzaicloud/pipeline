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
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// Store object that wraps up vault logical store
// nolint: gochecknoglobals
var Store *secretStore

// ErrSecretNotExists denotes 'Not Found' errors for secrets
// nolint: gochecknoglobals
var ErrSecretNotExists = fmt.Errorf("There's no secret with this ID")

// InitSecretStore initializes the global secret store.
func InitSecretStore(store secret.Store, types secret.TypeList) {
	Store = &secretStore{
		SecretStore: store,
		Types:       types,
	}
}

type secretStore struct {
	SecretStore secret.Store
	Types       secret.TypeList
}

// CreateSecretRequest param for secretStore.Store
// Only fields with `mapstructure` tag are getting written to Vault
type CreateSecretRequest struct {
	Name      string            `json:"name" binding:"required" mapstructure:"name"`
	Type      string            `json:"type" binding:"required" mapstructure:"type"`
	Values    map[string]string `json:"values" binding:"required" mapstructure:"values"`
	Tags      []string          `json:"tags,omitempty" mapstructure:"tags"`
	UpdatedBy string            `json:"updatedBy,omitempty" mapstructure:"updatedBy"`

	// Verify secret if the type has a verifier
	Verify bool `json:"-" mapstructure:"-"`
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
	if s.Type != validType {
		return MismatchError{
			SecretType: s.Type,
			ValidType:  validType,
		}
	}
	return nil
}

// GenerateSecretIDFromName generates a "unique by name per organization" id for Secrets
func GenerateSecretIDFromName(name string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(name)))
}

// GenerateSecretID generates a "unique by name per organization" id for Secrets
func GenerateSecretID(request *CreateSecretRequest) string {
	return GenerateSecretIDFromName(request.Name)
}

// DeleteByClusterUID Delete secrets by ClusterUID
func (ss *secretStore) DeleteByClusterUID(orgID uint, clusterUID string) error {
	if clusterUID == "" {
		return errors.New("clusterUID is empty")
	}

	log := log.WithFields(map[string]interface{}{"organization": orgID, "clusterUID": clusterUID})

	clusterUIDTag := clusterUIDTag(clusterUID)
	secrets, err := ss.List(orgID,
		&ListSecretsQuery{
			Tags: []string{clusterUIDTag},
		})
	if err != nil {
		log.Error(fmt.Sprintf("Error during list secrets: %s", err.Error()))
		return err
	}

	for _, s := range secrets {
		log := log.WithFields(map[string]interface{}{"secret": s.ID, "secretName": s.Name})
		err := ss.Delete(orgID, s.ID)
		if err != nil {
			log.Error(fmt.Sprintf("Error during delete secret: %s", err.Error()))
		}
		log.Info("Secret Deleted")
	}

	return nil
}

// Delete secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Delete(organizationID uint, secretID string) error {
	log.Debug("deleting secret", map[string]interface{}{
		"organizationId": organizationID,
		"secretId":       secretID,
	})

	s, err := ss.Get(organizationID, secretID)
	if err == ErrSecretNotExists { // Already deleted
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "Error during querying secret before deletion")
	}

	secretType := ss.Types.Type(s.Type)
	if secretType == nil {
		return errors.Errorf("wrong secret type: %s", s.Type)
	}

	if err := ss.SecretStore.Delete(context.Background(), organizationID, secretID); err != nil {
		return err
	}

	if ct, ok := secretType.(secret.CleanupType); ok {
		err := ct.Cleanup(organizationID, s.Values, s.Tags)
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

	secretType := ss.Types.Type(request.Type)
	if secretType == nil {
		return "", errors.Errorf("wrong secret type: %s", request.Type)
	}

	if gt, ok := secretType.(secret.GeneratorType); ok {
		complete, err := gt.ValidateNew(request.Values)
		if err != nil {
			return "", err
		}

		if !complete {
			values, err := gt.Generate(organizationID, request.Name, request.Values, request.Tags)
			if err != nil {
				return "", err
			}

			request.Values = values
		}
	} else {
		err := secretType.Validate(request.Values)
		if err != nil {
			return "", err
		}
	}

	if pt, ok := secretType.(secret.ProcessorType); ok {
		values, err := pt.Process(request.Values)
		if err != nil {
			return "", err
		}

		request.Values = values
	}

	if request.Verify {
		if vt, ok := secretType.(secret.VerifierType); ok {
			err := vt.Verify(request.Values)
			if err != nil {
				return "", err
			}
		}
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

	secretType := ss.Types.Type(request.Type)
	if secretType == nil {
		return errors.Errorf("wrong secret type: %s", request.Type)
	}

	err := secretType.Validate(request.Values)
	if err != nil {
		return err
	}

	if pt, ok := secretType.(secret.ProcessorType); ok {
		values, err := pt.Process(request.Values)
		if err != nil {
			return err
		}

		request.Values = values
	}

	if request.Verify {
		if vt, ok := secretType.(secret.VerifierType); ok {
			err := vt.Verify(request.Values)
			if err != nil {
				return err
			}
		}
	}

	log.Debug("updating secret", map[string]interface{}{
		"organizationId": organizationID,
		"secretId":       secretID,
	})

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
		log.Error(fmt.Sprintf("Error during checking secret: %s", err.Error()))
		return "", err
	} else if secret != nil {
		return secret.ID, nil
	} else {
		secretID, err = ss.Store(organizationID, value)
		if err != nil {
			log.Error(fmt.Sprintf("Error during storing secret: %s", err.Error()))
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
		log.Error(fmt.Sprintf("Error during checking secret: %s", err.Error()))
		return "", err
	} else if secret != nil {
		err := ss.Update(organizationID, secretID, value)
		if err != nil {
			log.Error(fmt.Sprintf("Error during updating secret: %s", err.Error()))
			return "", err
		}
	} else {
		secretID, err = ss.Store(organizationID, value)
		if err != nil {
			log.Error(fmt.Sprintf("Error during storing secret: %s", err.Error()))
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
		Version:   1,
		UpdatedAt: model.UpdatedAt,
		UpdatedBy: model.UpdatedBy,
	}, nil
}

// Retrieve secret by secret Name secret/orgs/:orgid:/:id: scope
func (ss *secretStore) GetByName(organizationID uint, name string) (*SecretItemResponse, error) {
	secretID := GenerateSecretIDFromName(name)

	return ss.Get(organizationID, secretID)
}

// List secret secret/orgs/:orgid:/ scope
func (ss *secretStore) List(orgid uint, query *ListSecretsQuery) ([]*SecretItemResponse, error) {
	log.Debug(fmt.Sprintf("Searching for secrets [orgid: %d, query: %#v]", orgid, query))

	if query.Type != "" {
		secretType := ss.Types.Type(query.Type)
		if secretType == nil {
			return nil, errors.Errorf("wrong secret type: %s", query.Type)
		}
	}

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
			log.Error(fmt.Sprintf("Error listing secrets: %s", err.Error()))

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
			Version:   1,
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

// Verify secret secret/orgs/:orgid:/:id: scope
func (ss *secretStore) Verify(organizationID uint, secretID string) error {
	s, err := ss.Get(organizationID, secretID)
	if err != nil {
		return err
	}

	secretType := ss.Types.Type(s.Type)
	if secretType == nil {
		return errors.Errorf("wrong secret type: %s", s.Type)
	}

	if vt, ok := secretType.(secret.VerifierType); ok {
		err := vt.Verify(s.Values)
		if err != nil {
			return err
		}
	}

	return nil
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

func (MismatchError) BadRequest() bool {
	return true
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

const clusterUIDTagName = "clusterUID"

func clusterUIDTag(clusterUID string) string {
	return fmt.Sprintf("%s:%s", clusterUIDTagName, clusterUID)
}
