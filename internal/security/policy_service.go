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

package anchore

import (
	"context"
	"strconv"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/common"
)

//PolicyService policy related operations
type PolicyService interface {
	UpdatePolicy(ctx context.Context, orgID uint, clusterID uint, policyID string, policyActivate pipeline.PolicyBundleActivate) error
}

type policyService struct {
	configService ConfigurationService
	secretStore   common.SecretStore
	logger        common.Logger
}

func NewPolicyService(configService ConfigurationService, store common.SecretStore, logger common.Logger) PolicyService {
	return policyService{
		configService: configService,
		secretStore:   store,
		logger:        logger.WithFields(map[string]interface{}{"policy-service": "y"}),
	}
}

func (p policyService) UpdatePolicy(ctx context.Context, orgID uint, clusterID uint, policyID string, policyActivate pipeline.PolicyBundleActivate) error {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "policyID": policyID}
	p.logger.Info("updating policy ...", fnCtx)

	anchoreClient, err := p.getAnchoreClient(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore client", fnCtx)

		return errors.WrapIf(err, "failed to get anchore client")
	}

	activate, err := strconv.ParseBool(policyActivate.Params.Active)
	if err != nil {
		p.logger.Debug("failed to parse activate param", fnCtx)

		return errors.WrapIf(err, "failed to parse activate param")
	}

	if err := anchoreClient.UpdatePolicy(ctx, policyID, activate); err != nil {
		p.logger.Debug("failure while updating policy", fnCtx)

		return errors.WrapIf(err, "failed to update policy")
	}

	p.logger.Info("policy successfully updated", fnCtx)
	return nil
}

// getAnchoreClient returns p rest client wrapper instance with the proper configuration
// todo this method may be extracted to p common place to be reused by other services
func (p policyService) getAnchoreClient(ctx context.Context, clusterID uint) (AnchoreClient, error) {
	cfg, err := p.configService.GetConfiguration(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failure while getting anchore configuration")

		return nil, errors.Wrap(err, "failed to get anchore configuration")
	}

	if !cfg.Enabled {
		p.logger.Debug("anchore service disabled")

		return nil, errors.NewWithDetails("anchore service disabled", "clusterID", clusterID)
	}

	if cfg.UserSecret != "" {
		p.logger.Debug("using custom anchore configuration")
		username, password, err := GetCustomAnchoreCredentials(ctx, p.secretStore, cfg.UserSecret, p.logger)
		if err != nil {
			p.logger.Debug("failed to decode secret values")

			return nil, errors.WrapIf(err, "failed to decode custom anchore user secret")
		}

		return NewAnchoreClient(username, password, cfg.Endpoint, p.logger), nil
	}

	userName := GetUserName(clusterID)
	password, err := GetUserSecret(ctx, p.secretStore, userName, p.logger)
	if err != nil {
		p.logger.Debug("failed to get user secret")

		return nil, errors.Wrap(err, "failed to get anchore configuration")
	}

	return NewAnchoreClient(userName, password, cfg.Endpoint, p.logger), nil
}
