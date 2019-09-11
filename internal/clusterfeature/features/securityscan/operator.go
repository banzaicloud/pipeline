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

package securityscan

import (
	"context"
	"encoding/json"
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/mitchellh/mapstructure"
)

const (
	securityScanChartVersion = ""
	securityScanChartName    = "banzaicloud-stable/anchore-policy-validator"
	securityScanNamespace    = "pipeline-system"
	securityScanRelease      = "anchore"
)

type featureOperator struct {
	clusterGetter  clusterfeatureadapter.ClusterGetter
	clusterService clusterfeature.ClusterService
	helmService    features.HelmService
	secretStore    features.SecretStore
	logger         common.Logger
}

func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	secretStore features.SecretStore,
	logger common.Logger,

) featureOperator {
	return featureOperator{
		clusterGetter:  clusterGetter,
		clusterService: clusterService,
		helmService:    helmService,
		secretStore:    secretStore,
		logger:         logger,
	}
}

func (op featureOperator) Apply(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": FeatureName})

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return err
	}

	if boundSpec.CustomAnchore.Enabled {
		// todo manage custom anchore (and possibly quit)
	}

	// default (pipeline hosted) anchore
	if !anchore.AnchoreEnabled {
		logger.Info("Anchore integration is not enabled.")
		return errors.NewWithDetails("default anchore is not enabled")
	}

	secretName, err := op.createAnchoreUserForCluster(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to create anchore user")
	}

	anchoreSecretID := secret.GenerateSecretIDFromName(secretName)
	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, anchoreSecretID)
	if err != nil {
		return errors.WrapWithDetails(err, "failed to get anchore secret", "user", secretName)
	}

	// todo cluster.SetSecurityScan(true) - set this

	values, err := op.processChartValues(anchoreUserSecret)
	if err != nil {
		return errors.WrapIf(err, "failed to assemble chart values")
	}

	if err = op.helmService.ApplyDeployment(ctx, clusterID,
		securityScanNamespace,
		securityScanChartName,
		securityScanRelease,
		values,
		securityScanChartVersion); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	// todo install all whitelist

	return nil
}

func (op featureOperator) Deactivate(ctx context.Context, clusterID uint) error {
	panic("implement me")
}

func (op featureOperator) Name() string {
	return FeatureName
}

// todo move this out to a common place as it's duplicated now (in the dns feature)
func (op featureOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cluster.GetOrganizationId())
	}
	return ctx, nil
}

func (op featureOperator) createAnchoreUserForCluster(ctx context.Context, clusterID uint) (string, error) {
	cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)

	anchoreUserName := fmt.Sprintf("%v-anchore-user", cluster.GetUID())

	// todo decouple anchore integration here
	_, err = anchore.SetupAnchoreUser(cluster.GetOrganizationId(), cluster.GetUID())
	if err != nil {
		return "", errors.WrapWithDetails(err, "error creating anchore user",
			"organization", cluster.GetOrganizationId(),
			"anchore-user", anchoreUserName)
	}

	return anchoreUserName, nil
}

func (op featureOperator) processChartValues(secretValues map[string]string) ([]byte, error) {

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(secretValues, &anchoreValues); err != nil {
		return nil, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	// todo pass configuration instead using these values
	anchoreValues.Host = anchore.AnchoreEndpoint

	securityScanCharValues := SecurityScanChartValues{
		Anchore:     anchoreValues,
		Affinity:    nil, // todo fill this
		Tolerations: nil, // todo fill this
	}

	values, err := json.Marshal(&securityScanCharValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal chart values")
	}

	return values, nil
}
