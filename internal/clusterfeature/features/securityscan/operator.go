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

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	securityScanChartVersion = "0.4.4"
	// todo read this from the chart possibly
	imageValidatorVersion = "0.3.6"

	// anchore version
	securityScanChartName = "banzaicloud-stable/anchore-policy-validator"
	securityScanNamespace = "pipeline-system"
	securityScanRelease   = "anchore"
)

type FeatureOperator struct {
	anchoreEnabled   bool
	anchoreEndpoint  string
	clusterGetter    clusterfeatureadapter.ClusterGetter
	clusterService   clusterfeature.ClusterService
	helmService      features.HelmService
	secretStore      features.SecretStore
	anchoreService   FeatureAnchoreService
	whiteListService FeatureWhiteListService
	namespaceService NamespaceService
	errorHandler     common.ErrorHandler
	logger           common.Logger
}

func MakeFeatureOperator(
	anchoreEnabled bool,
	anchoreEndpoint string,
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	secretStore features.SecretStore,
	anchoreService FeatureAnchoreService,
	featureWhitelistService FeatureWhiteListService,
	errorHandler common.ErrorHandler,
	logger common.Logger,

) FeatureOperator {
	return FeatureOperator{
		anchoreEnabled:   anchoreEnabled,
		anchoreEndpoint:  anchoreEndpoint,
		clusterGetter:    clusterGetter,
		clusterService:   clusterService,
		helmService:      helmService,
		secretStore:      secretStore,
		anchoreService:   anchoreService,
		whiteListService: featureWhitelistService,
		namespaceService: NewNamespacesService(clusterGetter, logger), // wired service
		errorHandler:     errorHandler,
		logger:           logger,
	}
}

// Name returns the name of the feature
func (op FeatureOperator) Name() string {
	return FeatureName
}

func (op FeatureOperator) Apply(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": FeatureName})
	logger.Info("start to apply feature")

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to apply feature")
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to apply feature")
	}

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return errors.WrapIf(err, "failed to apply feature")
	}

	var anchoreValues *AnchoreValues
	if boundSpec.CustomAnchore.Enabled {
		anchoreValues, err = op.getCustomAnchoreValues(ctx, boundSpec.CustomAnchore)
		if err != nil {
			return errors.WrapIf(err, "failed to get default anchore values")
		}
	} else {
		anchoreValues, err = op.getDefaultAnchoreValues(ctx, clusterID)
		if err != nil {
			return errors.WrapIf(err, "failed to get default anchore values")
		}
	}

	if err := op.setSecurityScan(ctx, clusterID, true); err != nil {
		return errors.WrapIf(err, "failed to set security scan flag on cluster")
	}

	values, err := op.processChartValues(ctx, clusterID, *anchoreValues)
	if err != nil {
		return errors.WrapIf(err, "failed to assemble chart values")
	}

	if err = op.helmService.ApplyDeployment(ctx, clusterID, securityScanNamespace, securityScanChartName, securityScanRelease,
		values, securityScanChartVersion); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	if len(boundSpec.ReleaseWhiteList) > 0 {
		if err = op.whiteListService.EnsureReleaseWhiteList(ctx, clusterID, boundSpec.ReleaseWhiteList); err != nil {
			return errors.WrapIf(err, "failed to install release white list")
		}
	}

	if boundSpec.WebhookConfig.Enabled {
		if err = op.configureWebHook(ctx, clusterID, boundSpec.WebhookConfig); err != nil {
			//  as agreed, we let the feature activation to succeed and log the errors
			op.errorHandler.Handle(ctx, err)
		}
	}
	return nil
}

func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to deactivate feature")
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to deactivate feature")
	}

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster by ID")
	}

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		op.logger.Debug("failed to bind the spec")

		return errors.WrapIf(err, "failed to apply feature")
	}

	if err := op.helmService.DeleteDeployment(ctx, clusterID, securityScanRelease); err != nil {
		return errors.WrapIfWithDetails(err, "failed to uninstall feature", "feature", FeatureName,
			"clusterID", clusterID)
	}

	if err := op.namespaceService.RemoveLabels(ctx, clusterID, boundSpec.WebhookConfig.Namespaces, []string{"scan"}); err != nil {

		// if the operation fails for some reason (eg. non-existent namespaces) we notice that and let the deactivation succeed
		op.logger.Warn("failed to delete namespace labels", map[string]interface{}{"clusterID": clusterID})
		return nil

	}

	if err := op.setSecurityScan(ctx, clusterID, false); err != nil {
		return errors.WrapIf(err, "failed to set security scan flag to false")
	}

	if !boundSpec.CustomAnchore.Enabled {
		if err = op.anchoreService.DeleteUser(ctx, cl.GetOrganizationId(), clusterID); err != nil {
			// deactivation succeeds even in case the generated anchore user is not deleted!
			op.logger.Warn("failed to delete the anchore user generated for the cluster", map[string]interface{}{"clusterID": clusterID})
			return nil
		}
	}

	return nil
}

func (op FeatureOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cl.GetOrganizationId())
	}
	return ctx, nil
}

func (op FeatureOperator) createAnchoreUserForCluster(ctx context.Context, clusterID uint) (string, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "error retrieving cluster")
	}

	userName, err := op.anchoreService.GenerateUser(ctx, cl.GetOrganizationId(), clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "error creating anchore user")
	}

	return userName, nil
}

func (op FeatureOperator) processChartValues(ctx context.Context, clusterID uint, anchoreValues AnchoreValues) ([]byte, error) {
	securityScanValues := SecurityScanChartValues{
		Anchore: anchoreValues,
	}

	values, err := json.Marshal(securityScanValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal chart values")
	}

	return values, nil
}

func (op FeatureOperator) getCustomAnchoreValues(ctx context.Context, customAnchore anchoreSpec) (*AnchoreValues, error) {
	if !customAnchore.Enabled { // this is already checked
		return nil, errors.NewWithDetails("custom anchore disabled")
	}

	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, customAnchore.SecretID)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "failed to get anchore secret", "secretId", customAnchore.SecretID)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return nil, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	anchoreValues.Host = customAnchore.Url

	return &anchoreValues, nil
}

func (op FeatureOperator) getDefaultAnchoreValues(ctx context.Context, clusterID uint) (*AnchoreValues, error) {

	// default (pipeline hosted) anchore
	if !op.anchoreEnabled {
		return nil, errors.NewWithDetails("default anchore is not enabled")
	}

	secretName, err := op.createAnchoreUserForCluster(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create anchore user")
	}

	anchoreSecretID := secret.GenerateSecretIDFromName(secretName)
	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, anchoreSecretID)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "failed to get anchore secret", "secretId", anchoreSecretID)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return nil, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	anchoreValues.Host = op.anchoreEndpoint

	return &anchoreValues, nil
}

// setSecurityScan temporary workaround for signaling the security scan enablement
func (op *FeatureOperator) setSecurityScan(ctx context.Context, clusterID uint, enabled bool) error {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	type securityScanFlagAwareCluster interface {
		SetSecurityScan(scan bool)
	}

	securityCluster := cl.(securityScanFlagAwareCluster)
	securityCluster.SetSecurityScan(enabled)

	return nil
}

func (op *FeatureOperator) configureWebHook(ctx context.Context, clusterID uint, whConfig webHookConfigSpec) error {

	const labelKey = "scan"
	securityScanLabels := map[string]string{
		"include": "scan",
		"exclude": "noscan",
	}

	labeMap := map[string]string{labelKey: securityScanLabels[whConfig.Selector]}

	if err := op.namespaceService.LabelNamespaces(ctx, clusterID, whConfig.Namespaces, labeMap); err != nil {
		return errors.WrapIf(err, "failed to label namespaces")
	}

	return nil
}
