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

package helmadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
)

type secretStore struct {
	secrets common.SecretStore
	logger  common.Logger
}

func NewSecretStore(store common.SecretStore, logger common.Logger) helm.SecretStore {
	return secretStore{
		secrets: store,
		logger:  logger,
	}
}

func (s secretStore) CheckPasswordSecret(ctx context.Context, secretID string) error {
	// todo validate the secret type too
	return s.secretExists(ctx, secretID)
}

func (s secretStore) CheckTLSSecret(ctx context.Context, secretID string) error {
	// todo validate the secret type too
	return s.secretExists(ctx, secretID)
}

func (s secretStore) secretExists(ctx context.Context, secretID string) error {
	if _, err := s.secrets.GetSecretValues(ctx, secretID); err != nil {
		return errors.WrapIf(err, "failed to retrieve secret values")
	}

	return nil
}
