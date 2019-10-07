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

package adapter

import (
	"context"
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/pke/workflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/brn"
)

type PasswordSecretStore struct {
	store SecretStore
}

type SecretStore interface {
	GetSecretValues(ctx context.Context, secretID string) (map[string]string, error)
}

func NewPasswordSecretStore(store SecretStore) PasswordSecretStore {
	return PasswordSecretStore{
		store: store,
	}
}

func (s PasswordSecretStore) GetSecret(ctx context.Context, orgID uint, secretID string) (workflow.PasswordSecret, error) {
	if !brn.IsBRN(secretID) {
		secretID = brn.ResourceName{
			Scheme:         brn.Scheme,
			OrganizationID: orgID,
			ResourceType:   brn.SecretResourceType,
			ResourceID:     secretID,
		}.String()
	}

	values, err := s.store.GetSecretValues(ctx, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get secret values")
	}
	ps, err := toPasswordSecret(values)
	return ps, errors.WrapIf(err, "failed to adapt password secret")
}

func toPasswordSecret(s map[string]string) (ps passwordSecret, err error) {
	ps.username, err = getSecretValue(s, secrettype.Username)
	if err != nil {
		return
	}
	ps.password, err = getSecretValue(s, secrettype.Password)
	if err != nil {
		return
	}
	return
}

func getSecretValue(s map[string]string, k string) (string, error) {
	v, ok := s[k]
	if !ok {
		return "", errors.New(fmt.Sprintf("%q key missing from secret", k))
	}
	return v, nil
}

type passwordSecret struct {
	username string
	password string
}

func (s passwordSecret) Username() string {
	return s.username
}

func (s passwordSecret) Password() string {
	return s.password
}
