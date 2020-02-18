// Copyright © 2020 Banzai Cloud
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

package secretadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"

	"github.com/banzaicloud/pipeline/internal/secret"
)

// NewVaultStore returns a new secret store backed by Vault.
func NewVaultStore(client *vault.Client, mountPath string) secret.Store {
	return vaultStore{
		client: client,

		mountPath: strings.Trim(mountPath, "/"),
	}
}

type vaultStore struct {
	client *vault.Client

	mountPath string
}

func (s vaultStore) Create(_ context.Context, organizationID uint, model secret.Model) error {
	path := s.secretDataPath(organizationID, model.ID)

	sort.Strings(model.Tags)

	data, err := secretData(0, model)
	if err != nil {
		return err
	}

	if _, err := s.client.RawClient().Logical().Write(path, data); err != nil {
		if strings.Contains(err.Error(), "check-and-set parameter did not match the current version") {
			return secret.AlreadyExistsError{
				OrganizationID: organizationID,
				SecretID:       model.ID,
			}
		}

		return errors.Wrap(err, "failed to store secret")
	}

	return nil
}

func (s vaultStore) Put(_ context.Context, organizationID uint, model secret.Model) error {
	path := s.secretDataPath(organizationID, model.ID)

	sort.Strings(model.Tags)

	var version int
	{ // Check if a secret on this path already exists
		vaultSecret, err := s.client.RawClient().Logical().Read(path)
		if err != nil {
			return errors.Wrap(err, "failed to check if secret exists")
		}

		if vaultSecret != nil {
			metadata := cast.ToStringMap(vaultSecret.Data["metadata"])
			v, _ := metadata["version"].(json.Number).Int64()

			version = int(v)
		}
	}

	data, err := secretData(version, model)
	if err != nil {
		return err
	}

	if _, err := s.client.RawClient().Logical().Write(path, data); err != nil {
		return errors.Wrap(err, "failed to store secret")
	}

	return nil
}

func (s vaultStore) Get(_ context.Context, organizationID uint, id string) (secret.Model, error) {
	path := s.secretDataPath(organizationID, id)

	vaultSecret, err := s.client.RawClient().Logical().Read(path)
	if err != nil {
		return secret.Model{}, errors.Wrap(err, "failed to read secret")
	}

	if vaultSecret == nil {
		return secret.Model{}, errors.WithStack(secret.NotFoundError{
			OrganizationID: organizationID,
			SecretID:       id,
		})
	}

	return parseSecret(id, vaultSecret)
}

func (s vaultStore) List(_ context.Context, organizationID uint) ([]secret.Model, error) {
	path := fmt.Sprintf("%s/metadata/orgs/%d", s.mountPath, organizationID)

	vaultSecretList, err := s.client.RawClient().Logical().List(path)
	if err != nil {
		return nil, err
	}

	if vaultSecretList == nil {
		return []secret.Model{}, nil
	}

	keys := cast.ToStringSlice(vaultSecretList.Data["keys"])

	models := make([]secret.Model, 0, len(keys))

	for _, key := range keys {
		vaultSecret, err := s.client.RawClient().Logical().Read(s.secretDataPath(organizationID, key))
		if err != nil {
			return nil, errors.WrapWithDetails(
				err, "failed to list secrets",
				"organizationId", organizationID,
				"secretId", key,
			)
		}

		if vaultSecret == nil { // Secret was removed?
			continue
		}

		model, err := parseSecret(key, vaultSecret)
		if err != nil {
			return nil, errors.WithDetails(
				err,
				"organizationId", organizationID,
				"secretId", key,
			)
		}

		models = append(models, model)
	}

	return models, nil
}

func (s vaultStore) Delete(_ context.Context, organizationID uint, id string) error {
	path := fmt.Sprintf("%s/metadata/orgs/%d/%s", s.mountPath, organizationID, id)

	if _, err := s.client.RawClient().Logical().Delete(path); err != nil {
		return errors.WrapWithDetails(
			err, "failed to delete secret",
			"organizationId", organizationID,
			"secretId", id,
		)
	}

	return nil
}

func (s vaultStore) secretDataPath(organizationID uint, secretID string) string {
	return fmt.Sprintf("%s/data/orgs/%d/%s", s.mountPath, organizationID, secretID)
}

func secretData(version int, model secret.Model) (map[string]interface{}, error) {
	values := map[string]interface{}{}

	if err := mapstructure.Decode(model, &values); err != nil {
		return nil, errors.WrapWithDetails(
			err, "failed to encode secret",
			"secretId", model.ID,
		)
	}

	return vault.NewData(version, map[string]interface{}{"value": values}), nil
}

func parseSecret(id string, vaultSecret *vaultapi.Secret) (secret.Model, error) {
	data := cast.ToStringMap(vaultSecret.Data["data"])
	metadata := cast.ToStringMap(vaultSecret.Data["metadata"])

	updatedAt, err := time.Parse(time.RFC3339, metadata["created_time"].(string))
	if err != nil {
		return secret.Model{}, errors.Wrap(err, "failed to parse update time")
	}

	model := secret.Model{
		ID:        id,
		UpdatedAt: updatedAt,
		Tags:      []string{},
	}

	if err := mapstructure.Decode(data["value"], &model); err != nil {
		return model, errors.Wrap(err, "failed to parse secret")
	}

	return model, nil
}
