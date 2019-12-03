// Copyright © 2019 Banzai Cloud
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

package restricted

import (
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
)

const (
	orgID = 19
)

// nolint: gochecknoglobals
var version = 1

func TestBlockingTags(t *testing.T) {

	cases := []struct {
		name    string
		request *secret.CreateSecretRequest
	}{
		{name: "readonly", request: &requestReadOnly},
		{name: "forbidden", request: &requestForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := restrictedSecretStore{
				secretStore: inMemorySecretStore{
					secrets: make(map[uint]map[string]secret.CreateSecretRequest),
				},
			}

			secretID, err := store.Store(orgID, tc.request)

			if err != nil {
				t.Errorf("error during storing readonly secret: %s", err.Error())
				t.FailNow()
			}

			err = store.Delete(orgID, secretID)
			if err == nil {
				t.Error("readonly secret deleted..")
				t.FailNow()

				tc.request.Tags = append(tc.request.Tags, "newtag")

				err = store.Update(orgID, secretID, tc.request)
				if err == nil {
					t.Error("readonly secret updated..")
					t.FailNow()
				}

				expErr := ReadOnlyError{SecretID: secretID}
				if !reflect.DeepEqual(err, expErr) {
					t.Errorf("expected error: %s, got: %s", expErr, err.Error())
					t.FailNow()
				}
			}
		})
	}

}

// nolint: gochecknoglobals
var (
	requestReadOnly = secret.CreateSecretRequest{
		Name: "readonly",
		Type: secrettype.Password,
		Values: map[string]string{
			"key": "value",
		},
		Tags: []string{
			secret.TagBanzaiReadonly,
		},
		Version:   version,
		UpdatedBy: "banzaiuser",
	}

	requestForbidden = secret.CreateSecretRequest{
		Name: "forbidden",
		Type: secrettype.Password,
		Values: map[string]string{
			"key": "value",
		},
		Tags:      secret.ForbiddenTags,
		Version:   version,
		UpdatedBy: "banzaiuser",
	}
)

type inMemorySecretStore struct {
	secrets map[uint]map[string]secret.CreateSecretRequest
}

func (ss inMemorySecretStore) Delete(orgID uint, secretID string) error {
	if os, ok := ss.secrets[orgID]; ok {
		delete(os, secretID)
	}
	return nil
}

func (ss inMemorySecretStore) Get(orgID uint, secretID string) (*secret.SecretItemResponse, error) {
	if os := ss.secrets[orgID]; os != nil {
		if s, ok := os[secretID]; ok {
			return &secret.SecretItemResponse{
				ID:        secretID,
				Name:      s.Name,
				Type:      s.Type,
				Values:    s.Values,
				Tags:      s.Tags,
				Version:   s.Version + 1,
				UpdatedBy: s.UpdatedBy,
			}, nil
		}
	}
	return nil, secret.ErrSecretNotExists
}

func (ss inMemorySecretStore) List(orgID uint, query *secret.ListSecretsQuery) ([]*secret.SecretItemResponse, error) {
	var list []*secret.SecretItemResponse
	for _, os := range ss.secrets {
		for secretID, s := range os {
			list = append(list, &secret.SecretItemResponse{
				ID:        secretID,
				Name:      s.Name,
				Type:      s.Type,
				Values:    s.Values,
				Tags:      s.Tags,
				Version:   s.Version + 1,
				UpdatedBy: s.UpdatedBy,
			})
		}
	}
	return list, nil
}

func (ss inMemorySecretStore) Store(orgID uint, request *secret.CreateSecretRequest) (string, error) {
	os := ss.secrets[orgID]
	if os == nil {
		os = make(map[string]secret.CreateSecretRequest)
		ss.secrets[orgID] = os
	}

	secretID := secret.GenerateSecretID(request)
	os[secretID] = *request

	return secretID, nil
}

func (ss inMemorySecretStore) Update(orgID uint, secretID string, request *secret.CreateSecretRequest) error {
	if os := ss.secrets[orgID]; os != nil {
		if _, ok := os[secretID]; ok {
			os[secretID] = *request
			return nil
		}
	}
	return secret.ErrSecretNotExists
}
