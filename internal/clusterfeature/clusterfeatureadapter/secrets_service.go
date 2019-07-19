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
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/logur"
)

type secretsService struct {
	logger logur.Logger
}

func (s *secretsService) GetSecretValues(ctx context.Context, secretName string, orgID uint) (interface{}, error) {

	s.logger.Info("resolving secret ...", map[string]interface{}{"name": secretName, "orgID": orgID})
	secret, err := secret.Store.GetByName(orgID, secretName)
	if err != nil {
		s.logger.Debug("failed to get secret", map[string]interface{}{"name": secretName, "orgID": orgID})

		return nil, errors.WrapIf(err, "failed to get secret")
	}

	s.logger.Info("secret resolved", map[string]interface{}{"name": secretName, "orgID": orgID})
	return secret.Values, nil

}

func NewSecretsService(logger logur.Logger) clusterfeature.SecretsService {

	return &secretsService{
		logger: logur.WithFields(logger, map[string]interface{}{"secrets-service": "comp"}),
	}
}
