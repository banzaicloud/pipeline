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
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/mitchellh/mapstructure"
	"k8s.io/api/core/v1"
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

	// todo cluster.SetSecurityScan(true) - set this
	values, err := op.processChartValues(ctx, clusterID, secretName)
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
		cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cl.GetOrganizationId())
	}
	return ctx, nil
}

func (op featureOperator) createAnchoreUserForCluster(ctx context.Context, clusterID uint) (string, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "error retrieving cluster")
	}

	anchoreUserName := fmt.Sprintf("%v-anchore-user", cl.GetUID())

	// todo decouple anchore integration here
	if _, err = anchore.SetupAnchoreUser(cl.GetOrganizationId(), cl.GetUID()); err != nil {
		return "", errors.WrapWithDetails(err, "error creating anchore user", "organization",
			cl.GetOrganizationId(), "anchore-user", anchoreUserName)
	}

	return anchoreUserName, nil
}

func (op featureOperator) getDefaultValues(ctx context.Context, clusterID uint) (*SecurityScanChartValues, error) {

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	return getDefaultValues(cl), nil
}

func getDefaultValues(cl clusterfeatureadapter.Cluster) *SecurityScanChartValues {
	chartValues := new(SecurityScanChartValues)

	if headNodeAffinity := cluster.GetHeadNodeAffinity(cl); headNodeAffinity != (v1.Affinity{}) {
		chartValues.Affinity = &headNodeAffinity
	}

	chartValues.Tolerations = cluster.GetHeadNodeTolerations()

	return chartValues
}

func (op featureOperator) processChartValues(ctx context.Context, clusterID uint, secretName string) ([]byte, error) {
	securityScanValues, err := op.getDefaultValues(ctx, clusterID)

	anchoreSecretID := secret.GenerateSecretIDFromName(secretName)
	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, anchoreSecretID)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "failed to get anchore secret", "user", secretName)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return nil, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	// todo pass configuration instead using these values
	anchoreValues.Host = anchore.AnchoreEndpoint
	securityScanValues.Anchore = anchoreValues

	values, err := json.Marshal(securityScanValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal chart values")
	}

	return values, nil

}
