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

package clusterfeatureadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret"
	secret2 "github.com/banzaicloud/pipeline/secret"
)

type SecretService interface {
	GetSecretByID(ctx context.Context, clusterID uint, secretID string) (map[string]string, error)
	GetSecretByName(ctx context.Context, clusterID uint, secretName string) (map[string]string, error)
}

type clusterSecretsService struct {
	clusterService ClusterGetter
	secretStore    secret.SecretStore

	logger common.Logger
}

func NewClusterSecretsStore(getter ClusterGetter, store secret.SecretStore, logger common.Logger) clusterfeature.ClusterSecretStore {

	return &clusterSecretsService{

		clusterService: getter,
		secretStore:    store,
		logger:         logger,
	}
}

func (s *clusterSecretsService) GetSecret(ctx context.Context, clusterID uint, secretID string) (map[string]string, error) {

	cluster, err := s.clusterService.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		s.logger.Debug("failed to retrieve cluster by id")

		return nil, errors.WrapIf(err, "failed to retrieve cluster by ID")
	}

	secretResponse, err := s.secretStore.Get(cluster.GetOrganizationId(), secretID)
	if err != nil {
		s.logger.Debug("failed to retrieve secret by id")

		return nil, errors.WrapIf(err, "failed to retrieve secret by ID")
	}

	return secretResponse.Values, nil
}

func (s *clusterSecretsService) GetSecretByName(ctx context.Context, clusterID uint, secretName string) (map[string]string, error) {

	return s.GetSecret(ctx, clusterID, secret2.GenerateSecretIDFromName(secretName))
}
