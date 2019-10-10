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

type PolicyService interface {
	ListPolicies(ctx context.Context, orgID uint, clusterID uint) (interface{}, error)
	GetPolicy(ctx context.Context, orgID uint, clusterID uint, policyID string) (*pipeline.PolicyBundleRecord, error)
	CreatePolicy(ctx context.Context, orgID uint, clusterID uint, policy pipeline.PolicyBundleRecord) (interface{}, error)
	DeletePolicy(ctx context.Context, orgID uint, clusterID uint, policyID string) error
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
		logger:        logger,
	}
}

func (p policyService) ListPolicies(ctx context.Context, orgID uint, clusterID uint) (interface{}, error) {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID}
	p.logger.Info("retrieving policies ...", fnCtx)

	anchoreClient, err := p.getAnchoreClient(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore client", fnCtx)

		return nil, err
	}

	policyList, err := anchoreClient.ListPolicies(ctx)
	if err != nil {
		p.logger.Debug("failure while retrieving policies", fnCtx)

		return nil, errors.WrapIf(err, "failure while retrieving policies")
	}

	p.logger.Info("policies successfully retrieved", fnCtx)
	return policyList, nil
}

func (p policyService) GetPolicy(ctx context.Context, orgID uint, clusterID uint, policyID string) (*pipeline.PolicyBundleRecord, error) {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "policyID": policyID}
	p.logger.Info("retrieving policy ...", fnCtx)

	anchoreClient, err := p.getAnchoreClient(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore client", fnCtx)

		return nil, err
	}

	policyItem, err := anchoreClient.GetPolicy(ctx, policyID)
	if err != nil {
		p.logger.Debug("failure while retrieving policy", fnCtx)

		return nil, errors.WrapIf(err, "failure while retrieving policy")
	}

	p.logger.Info("policies successfully retrieved", fnCtx)
	return policyItem, nil
}

func (p policyService) CreatePolicy(ctx context.Context, orgID uint, clusterID uint, policy pipeline.PolicyBundleRecord) (interface{}, error) {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "policy": policy}
	p.logger.Info("creating policy ...", fnCtx)

	anchoreClient, err := p.getAnchoreClient(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore client", fnCtx)

		return nil, err
	}

	policyItem, err := anchoreClient.CreatePolicy(ctx, policy)
	if err != nil {
		p.logger.Debug("failure while retrieving policy", fnCtx)

		return nil, errors.WrapIf(err, "failure while retrieving policy")
	}

	p.logger.Info("policies successfully retrieved", fnCtx)
	return policyItem, nil
}

func (p policyService) DeletePolicy(ctx context.Context, orgID uint, clusterID uint, policyID string) error {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "policyID": policyID}
	p.logger.Info("deleting policy ...", fnCtx)

	anchoreClient, err := p.getAnchoreClient(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore client", fnCtx)

		return errors.WrapIf(err, "failed to get anchore client")
	}

	if err := anchoreClient.DeletePolicy(ctx, policyID); err != nil {
		p.logger.Debug("failure while deleting policy", fnCtx)

		return errors.WrapIf(err, "failure while deleting policy")
	}

	p.logger.Info("policies successfully deleted", fnCtx)
	return nil
}

func (p policyService) UpdatePolicy(ctx context.Context, orgID uint, clusterID uint, policyID string, policyActivate pipeline.PolicyBundleActivate) error {
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID, "policyID": policyID}
	p.logger.Info("updating policy ...", fnCtx)

	anchoreClient, err := p.getAnchoreClient(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore client", fnCtx)

		return errors.WrapIf(err, "failed to get anchore client")
	}

	policy, err := anchoreClient.GetPolicy(ctx, policyID)
	if err != nil {
		p.logger.Debug("failed to retrieve policy to update", fnCtx)

		return errors.WrapIf(err, "failed to retrieve policy to update")
	}

	if activate, _ := strconv.ParseBool(policyActivate.Params.Active); activate {
		policy.Active = true
	}

	if err := anchoreClient.UpdatePolicy(ctx, policyID, *policy); err != nil {
		p.logger.Debug("failure while updating policy", fnCtx)

		return errors.WrapIf(err, "failure while updating policy")
	}

	p.logger.Info("policys successfully updated", fnCtx)
	return nil
}

// getAnchoreClient returns a rest client wrapper instance with the proper configuration
func (p policyService) getAnchoreClient(ctx context.Context, clusterID uint) (AnchoreClient, error) {
	cfg, err := p.configService.GetConfiguration(ctx, clusterID)
	if err != nil {
		p.logger.Debug("failed to get anchore configuration")

		return nil, errors.Wrap(err, "failed to get anchore configuration")
	}

	if !cfg.Enabled {
		p.logger.Debug("anchore service disabled")

		return nil, errors.NewWithDetails("anchore service disabled", "clusterID", clusterID)
	}

	userName := getUserName(clusterID)
	password, err := getUserSecret(ctx, p.secretStore, userName, p.logger)
	if err != nil {
		p.logger.Debug("failed to get user secret")

		return nil, errors.Wrap(err, "failed to get anchore configuration")
	}

	return NewAnchoreClient(userName, password, cfg.Endpoint, p.logger), nil
}
