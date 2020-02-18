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
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/suite"

	"github.com/banzaicloud/pipeline/internal/secret"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func testVaultStore(t *testing.T) {
	suite.Run(t, new(VaultStoreTestSuite))
}

const mountPath = "testsecret"

type VaultStoreTestSuite struct {
	suite.Suite

	client    *vault.Client
	mountPath string

	store secret.Store
}

func (s *VaultStoreTestSuite) SetupSuite() {
	client, err := vault.NewClient("")
	s.Require().NoError(err)

	s.mountPath = fmt.Sprintf("%s-%d%d%d", mountPath, rand.Intn(30), rand.Intn(30), rand.Intn(30))

	err = client.RawClient().Sys().Mount(s.mountPath, &vaultapi.MountInput{
		Type:        "kv",
		Description: "Mount point for secret integration tests",
		Options: map[string]string{
			"version": "2",
		},
	})
	s.Require().NoError(err)

	s.client = client
}

func (s *VaultStoreTestSuite) TearDownSuite() {
	err := s.client.RawClient().Sys().Unmount(s.mountPath)
	s.Require().NoError(err)

	s.client.Close()
}

func (s *VaultStoreTestSuite) SetupTest() {
	s.store = NewVaultStore(s.client, s.mountPath)
}

func (s *VaultStoreTestSuite) TestCreate() {
	model := secret.Model{
		ID:   "created-secret-id",
		Name: "created-secret-name",
		Type: "example",
		Values: map[string]string{
			"key": "value",
		},
		Tags:      []string{"tag:value"},
		UpdatedBy: "user",
	}

	err := s.store.Create(context.Background(), 1, model)
	s.Require().NoError(err)

	vaultSecret, err := s.client.RawClient().Logical().Read(fmt.Sprintf("%s/data/orgs/1/created-secret-id", s.mountPath))
	s.Require().NoError(err)

	expected := map[string]interface{}{
		"name":      model.Name,
		"type":      model.Type,
		"values":    map[string]interface{}{"key": model.Values["key"]},
		"tags":      []interface{}{model.Tags[0]},
		"updatedBy": "user",
	}

	s.Assert().EqualValues(expected, vaultSecret.Data["data"].(map[string]interface{})["value"])
}

func (s *VaultStoreTestSuite) TestCreate_AlreadyExists() {
	_, err := s.client.RawClient().Logical().Write(
		fmt.Sprintf("%s/data/orgs/1/already-existing-secret-id", s.mountPath),
		vault.NewData(0, map[string]interface{}{
			"value": map[string]interface{}{
				"name":      "already-existing-secret-name",
				"type":      "example",
				"values":    map[string]interface{}{"key": "value"},
				"tags":      []interface{}{"tag:value"},
				"updatedBy": "user",
			},
		}),
	)
	s.Require().NoError(err)

	model := secret.Model{
		ID:   "already-existing-secret-id",
		Name: "already-existing-secret-name",
		Type: "example",
		Values: map[string]string{
			"key": "value",
		},
		Tags:      []string{"tag:value"},
		UpdatedBy: "user",
	}

	err = s.store.Create(context.Background(), 1, model)
	s.Require().Error(err)

	var alreadyExistsErr secret.AlreadyExistsError
	if s.Assert().True(errors.As(err, &alreadyExistsErr)) {
		s.Assert().Equal(uint(1), alreadyExistsErr.OrganizationID)
		s.Assert().Equal("already-existing-secret-id", alreadyExistsErr.SecretID)
	}
}

func (s *VaultStoreTestSuite) TestPut() {
	_, err := s.client.RawClient().Logical().Write(
		fmt.Sprintf("%s/data/orgs/1/updated-secret-id", s.mountPath),
		vault.NewData(0, map[string]interface{}{
			"value": map[string]interface{}{
				"name":      "already-existing-secret-name",
				"type":      "example",
				"values":    map[string]interface{}{"key": "value"},
				"tags":      []interface{}{"tag:value"},
				"updatedBy": "user",
			},
		}),
	)
	s.Require().NoError(err)

	model := secret.Model{
		ID:   "updated-secret-id",
		Name: "updated-secret-name",
		Type: "example",
		Values: map[string]string{
			"key": "value2",
		},
		Tags:      []string{"tag:value2"},
		UpdatedBy: "user",
	}

	err = s.store.Put(context.Background(), 1, model)
	s.Require().NoError(err)

	vaultSecret, err := s.client.RawClient().Logical().Read(fmt.Sprintf("%s/data/orgs/1/updated-secret-id", s.mountPath))
	s.Require().NoError(err)

	expected := map[string]interface{}{
		"name":      model.Name,
		"type":      model.Type,
		"values":    map[string]interface{}{"key": model.Values["key"]},
		"tags":      []interface{}{model.Tags[0]},
		"updatedBy": "user",
	}

	s.Assert().Equal(expected, vaultSecret.Data["data"].(map[string]interface{})["value"])
}

func (s *VaultStoreTestSuite) TestPut_Create() {
	model := secret.Model{
		ID:   "updated-secret-id",
		Name: "updated-secret-name",
		Type: "example",
		Values: map[string]string{
			"key": "value2",
		},
		Tags:      []string{"tag:value2"},
		UpdatedBy: "user",
	}

	err := s.store.Put(context.Background(), 1, model)
	s.Require().NoError(err)

	vaultSecret, err := s.client.RawClient().Logical().Read(fmt.Sprintf("%s/data/orgs/1/updated-secret-id", s.mountPath))
	s.Require().NoError(err)

	expected := map[string]interface{}{
		"name":      model.Name,
		"type":      model.Type,
		"values":    map[string]interface{}{"key": model.Values["key"]},
		"tags":      []interface{}{model.Tags[0]},
		"updatedBy": "user",
	}

	s.Assert().Equal(expected, vaultSecret.Data["data"].(map[string]interface{})["value"])
}

func (s *VaultStoreTestSuite) TestGet() {
	_, err := s.client.RawClient().Logical().Write(
		fmt.Sprintf("%s/data/orgs/1/get-secret-id", s.mountPath),
		vault.NewData(0, map[string]interface{}{
			"value": map[string]interface{}{
				"name":      "get-secret-name",
				"type":      "example",
				"values":    map[string]interface{}{"key": "value"},
				"tags":      []interface{}{"tag:value"},
				"updatedBy": "user",
			},
		}),
	)
	s.Require().NoError(err)

	expected := secret.Model{
		ID:   "get-secret-id",
		Name: "get-secret-name",
		Type: "example",
		Values: map[string]string{
			"key": "value",
		},
		Tags:      []string{"tag:value"},
		UpdatedBy: "user",
	}

	actual, err := s.store.Get(context.Background(), 1, "get-secret-id")
	s.Require().NoError(err)

	// TODO: fix this test (if possible)?
	actual.UpdatedAt = time.Time{}

	s.Assert().Equal(expected, actual)
}

func (s *VaultStoreTestSuite) TestGet_NotFound() {
	_, err := s.store.Get(context.Background(), 1, "not-found-secret-id")
	s.Require().Error(err)

	var notFoundErr secret.NotFoundError
	if s.Assert().True(errors.As(err, &notFoundErr)) {
		s.Assert().Equal(uint(1), notFoundErr.OrganizationID)
		s.Assert().Equal("not-found-secret-id", notFoundErr.SecretID)
	}
}

func (s *VaultStoreTestSuite) TestList() {
	_, err := s.client.RawClient().Logical().Write(
		fmt.Sprintf("%s/data/orgs/2/list-secret-id", s.mountPath),
		vault.NewData(0, map[string]interface{}{
			"value": map[string]interface{}{
				"name":      "list-secret-name",
				"type":      "example",
				"values":    map[string]interface{}{"key": "value"},
				"tags":      []interface{}{"tag:value"},
				"updatedBy": "user",
			},
		}),
	)
	s.Require().NoError(err)

	expected := []secret.Model{
		{
			ID:   "list-secret-id",
			Name: "list-secret-name",
			Type: "example",
			Values: map[string]string{
				"key": "value",
			},
			Tags:      []string{"tag:value"},
			UpdatedBy: "user",
		},
	}

	actual, err := s.store.List(context.Background(), 2)
	s.Require().NoError(err)

	// TODO: fix this test (if possible)?
	actual[0].UpdatedAt = time.Time{}

	s.Assert().Equal(expected, actual)
}

func (s *VaultStoreTestSuite) TestDelete() {
	_, err := s.client.RawClient().Logical().Write(
		fmt.Sprintf("%s/data/orgs/1/delete-secret-id", s.mountPath),
		vault.NewData(0, map[string]interface{}{
			"value": map[string]interface{}{
				"name":      "delete-secret-name",
				"type":      "example",
				"values":    map[string]interface{}{"key": "value"},
				"tags":      []interface{}{"tag:value"},
				"updatedBy": "user",
			},
		}),
	)
	s.Require().NoError(err)

	err = s.store.Delete(context.Background(), 1, "delete-secret-id")
	s.Require().NoError(err)

	vaultSecret, err := s.client.RawClient().Logical().Read(fmt.Sprintf("%s/data/orgs/1/delete-secret-id", s.mountPath))
	s.Require().NoError(err)

	s.Assert().Nil(vaultSecret)
}

func (s *VaultStoreTestSuite) TestDelete_Idempotent() {
	err := s.store.Delete(context.Background(), 1, "delete-idempotent-secret-id")
	s.Require().NoError(err)
}
