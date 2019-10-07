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
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

type PasswordSecretStore struct {
	store SecretStore
}

type SecretStore interface {
	Get(orgID uint, secretID string) (*secret.SecretItemResponse, error)
}

func MakePasswordSecretStore(store SecretStore) PasswordSecretStore {
	return PasswordSecretStore{
		store: store,
	}
}

func (s PasswordSecretStore) GetSecret(ctx context.Context, orgID uint, secretID string) (workflow.PasswordSecret, error) {
	sir, err := s.store.Get(orgID, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get secret")
	}
	ps, err := toPasswordSecret(sir)
	return ps, errors.WrapIf(err, "failed to adapt password secret")
}

func toPasswordSecret(s *secret.SecretItemResponse) (ps passwordSecret, err error) {
	if s.Type != pkgSecret.PasswordSecretType {
		return ps, errors.New(fmt.Sprintf("secret must have type %q not %q", pkgSecret.PasswordSecretType, s.Type))
	}
	ps.username = s.GetValue(pkgSecret.Username)
	ps.password = s.GetValue(pkgSecret.Password)
	return
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
