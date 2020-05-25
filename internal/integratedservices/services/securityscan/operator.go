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

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
)

const (
	// the label key on the namespaces that is watched by the webhook
	labelKey = "scan"

	selectedAllStar = "*"
	selectorInclude = "include"
	selectorExclude = "exclude"
)

type IntegratedServiceOperator struct {
	config           Config
	clusterGetter    integratedserviceadapter.ClusterGetter
	clusterService   integratedservices.ClusterService
	helmService      services.HelmService
	secretStore      services.SecretStore
	anchoreService   IntegratedServiceAnchoreService
	whiteListService IntegratedServiceWhiteListService
	namespaceService NamespaceService
	errorHandler     common.ErrorHandler
	logger           common.Logger
}

func MakeIntegratedServiceOperator(
	config Config,
	clusterGetter integratedserviceadapter.ClusterGetter,
	clusterService integratedservices.ClusterService,
	helmService services.HelmService,
	secretStore services.SecretStore,
	anchoreService IntegratedServiceAnchoreService,
	integratedServiceWhitelistService IntegratedServiceWhiteListService,
	errorHandler common.ErrorHandler,
	logger common.Logger,

) IntegratedServiceOperator {
	return IntegratedServiceOperator{
		config:           config,
		clusterGetter:    clusterGetter,
		clusterService:   clusterService,
		helmService:      helmService,
		secretStore:      secretStore,
		anchoreService:   anchoreService,
		whiteListService: integratedServiceWhitelistService,
		namespaceService: NewNamespacesService(clusterGetter, logger), // wired service
		errorHandler:     errorHandler,
		logger:           logger,
	}
}

// Name returns the name of the integrated service
func (op IntegratedServiceOperator) Name() string {
	return IntegratedServiceName
}

func (op IntegratedServiceOperator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "integrated service": IntegratedServiceName})
	logger.Info("start to apply integrated service")

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to apply integrated service")
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to apply integrated service")
	}

	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return errors.WrapIf(err, "failed to apply integrated service")
	}

	var anchoreValues AnchoreValues
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

	values, err := assembleChartValues(anchoreValues, boundSpec.WebhookConfig)
	if err != nil {
		return errors.WrapIf(err, "failed to assemble chart values")
	}

	if err = op.helmService.ApplyDeployment(ctx, clusterID, op.config.Webhook.Namespace, op.config.Webhook.Chart, op.config.Webhook.Release,
		values, op.config.Webhook.Version); err != nil {
		return errors.WrapIf(err, "failed to deploy integrated service")
	}

	if len(boundSpec.ReleaseWhiteList) > 0 {
		if err = op.whiteListService.EnsureReleaseWhiteList(ctx, clusterID, boundSpec.ReleaseWhiteList); err != nil {
			return errors.WrapIf(err, "failed to install release white list")
		}
	}

	if boundSpec.WebhookConfig.Enabled {
		if err = op.applyLabelsForSecurityScan(ctx, clusterID, boundSpec.WebhookConfig); err != nil {
			//  as agreed, we let the integrated service activation to succeed and log the errors
			op.errorHandler.HandleContext(ctx, err)
		}
	}
	return nil
}

func (op IntegratedServiceOperator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to deactivate integrated service")
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to deactivate integrated service")
	}

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster by ID")
	}

	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		op.logger.Debug("failed to bind the spec")

		return errors.WrapIf(err, "failed to apply integrated service")
	}

	if err := op.helmService.DeleteDeployment(ctx, clusterID, op.config.Webhook.Release, op.config.Webhook.Namespace); err != nil {
		return errors.WrapIfWithDetails(err, "failed to uninstall integrated service", "integrated service", IntegratedServiceName,
			"clusterID", clusterID)
	}

	if err := op.namespaceService.CleanupLabels(ctx, clusterID, []string{labelKey}); err != nil {
		// if the operation fails for some reason (eg. non-existent namespaces) we notice that and let the deactivation succeed
		op.logger.Warn("failed to delete namespace labels", map[string]interface{}{"clusterID": clusterID})
		op.errorHandler.HandleContext(ctx, err)

		return nil
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

func (op IntegratedServiceOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cl.GetOrganizationId())
	}
	return ctx, nil
}

func (op IntegratedServiceOperator) createAnchoreUserForCluster(ctx context.Context, clusterID uint) (string, error) {
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

// assembleChartValues is in charge to assemble the values json for the chart based on the input and configuration
func assembleChartValues(anchoreValues AnchoreValues, webhookConfigSpec webHookConfigSpec) ([]byte, error) {
	chartValues := webhookConfigSpec.GetValues()
	chartValues.ExternalAnchore = &anchoreValues

	valuesBytes, err := json.Marshal(chartValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal chart values")
	}

	return valuesBytes, nil
}

func (op IntegratedServiceOperator) getCustomAnchoreValues(ctx context.Context, customAnchore anchoreSpec) (AnchoreValues, error) {
	if !customAnchore.Enabled { // this is already checked
		return AnchoreValues{}, errors.NewWithDetails("custom anchore disabled")
	}

	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, customAnchore.SecretID)
	if err != nil {
		return AnchoreValues{}, errors.WrapWithDetails(err, "failed to get anchore secret", "secretId", customAnchore.SecretID)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return AnchoreValues{}, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	anchoreValues.Host = customAnchore.Url
	anchoreValues.Insecure = customAnchore.Insecure

	return anchoreValues, nil
}

func (op IntegratedServiceOperator) getDefaultAnchoreValues(ctx context.Context, clusterID uint) (AnchoreValues, error) {
	// default (pipeline hosted) anchore
	if !op.config.Anchore.Enabled {
		return AnchoreValues{}, errors.NewWithDetails("default anchore is not enabled")
	}

	secretName, err := op.createAnchoreUserForCluster(ctx, clusterID)
	if err != nil {
		return AnchoreValues{}, errors.WrapIf(err, "failed to create anchore user")
	}

	anchoreSecretID := secret.GenerateSecretIDFromName(secretName)
	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, anchoreSecretID)
	if err != nil {
		return AnchoreValues{}, errors.WrapWithDetails(err, "failed to get anchore secret", "secretId", anchoreSecretID)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return AnchoreValues{}, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	anchoreValues.Host = op.config.Anchore.Endpoint
	anchoreValues.Insecure = op.config.Anchore.Insecure

	return anchoreValues, nil
}

// performs namespace labeling based on the provided input
func (op *IntegratedServiceOperator) applyLabelsForSecurityScan(ctx context.Context, clusterID uint, whConfig webHookConfigSpec) error {
	// possible label values that are used to make decisions by the webhook
	securityScanLabels := map[string]string{
		selectorInclude: "scan",
		selectorExclude: "noscan",
	}

	// remove all scan related labels from all namespaces
	if err := op.namespaceService.CleanupLabels(ctx, clusterID, []string{labelKey}); err != nil {
		// log the error and continue!
		op.errorHandler.HandleContext(ctx, err)
	}

	// these namespaces must always be excluded
	excludedNamespaces := []string{op.config.PipelineNamespace, "kube-system"}
	defaultExclusionMap := map[string]string{labelKey: securityScanLabels[selectorExclude]}

	if err := op.namespaceService.LabelNamespaces(ctx, clusterID, excludedNamespaces, defaultExclusionMap); err != nil {
		// log the error and continue!
		op.errorHandler.HandleContext(ctx, err)
	}

	if whConfig.Selector == selectorInclude && whConfig.allNamespaces() {
		// this setup corresponds to the default configuration, do nothing
		op.logger.Info("all namespaces are subject for security scan")
		return nil
	}

	// select the labels to be applied
	labeMap := map[string]string{labelKey: securityScanLabels[whConfig.Selector]}

	if err := op.namespaceService.LabelNamespaces(ctx, clusterID, whConfig.Namespaces, labeMap); err != nil {
		return errors.WrapIf(err, "failed to label namespaces")
	}

	return nil
}
