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

package monitoring

import (
	"context"
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

// FeatureManager implements the Monitoring feature manager
type FeatureManager struct {
	clusterGetter    clusterfeatureadapter.ClusterGetter
	secretStore      features.SecretStore
	endpointsService endpoints.EndpointService
	helmService      features.HelmService
	config           Configuration
	logger           common.Logger
}

func MakeFeatureManager(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	secretStore features.SecretStore,
	endpointsService endpoints.EndpointService,
	helmService features.HelmService,
	config Configuration,
	logger common.Logger,
) FeatureManager {
	return FeatureManager{
		clusterGetter:    clusterGetter,
		secretStore:      secretStore,
		endpointsService: endpointsService,
		helmService:      helmService,
		config:           config,
		logger:           logger,
	}
}

// Name returns the feature's name
func (FeatureManager) Name() string {
	return featureName
}

// GetOutput returns the Monitoring feature's output
func (m FeatureManager) GetOutput(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureOutput, error) {
	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return nil, clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	cluster, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get K8S config")
	}

	endpoints, err := m.endpointsService.List(kubeConfig, prometheusOperatorReleaseName)
	if err != nil {
		m.logger.Warn(fmt.Sprintf("failed to list endpoints: %s", err.Error()))
	}

	pushgatewayDeployment, err := m.helmService.GetDeployment(ctx, clusterID, prometheusPushgatewayReleaseName)
	if err != nil {
		m.logger.Warn(fmt.Sprintf("failed to get pushgateway details: %s", err.Error()))
	}

	operatorDeployment, err := m.helmService.GetDeployment(ctx, clusterID, prometheusOperatorReleaseName)
	if err != nil {
		m.logger.Warn(fmt.Sprintf("failed to get deployment details: %s", err.Error()))
	}

	var operatorValues map[string]interface{}
	if operatorDeployment != nil {
		operatorValues = operatorDeployment.Values
	}

	var pushgatewayValues map[string]interface{}
	if pushgatewayDeployment != nil {
		pushgatewayValues = pushgatewayDeployment.Values
	}

	chartVersion := m.config.operator.chartVersion
	pipelineSystemNamespace := m.config.pipelineSystemNamespace
	out := clusterfeature.FeatureOutput{
		"grafana":      m.getComponentOutput(ctx, clusterID, newGrafanaOutputHelper(kubeConfig, boundSpec), endpoints, pipelineSystemNamespace, prometheusOperatorReleaseName, operatorValues),
		"prometheus":   m.getComponentOutput(ctx, clusterID, newPrometheusOutputHelper(kubeConfig, boundSpec), endpoints, pipelineSystemNamespace, prometheusOperatorReleaseName, operatorValues),
		"alertmanager": m.getComponentOutput(ctx, clusterID, newAlertmanagerOutputHelper(kubeConfig, boundSpec), endpoints, pipelineSystemNamespace, prometheusOperatorReleaseName, operatorValues),
		"pushgateway":  m.getComponentOutput(ctx, clusterID, newPushgatewayOutputHelper(kubeConfig, boundSpec), endpoints, pipelineSystemNamespace, prometheusPushgatewayReleaseName, pushgatewayValues),
		"prometheusOperator": map[string]interface{}{
			"version": chartVersion,
		},
	}

	return out, nil
}

// ValidateSpec validates a Monitoring feature specification
func (FeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	if err := boundSpec.Validate(); err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	return nil
}

// PrepareSpec makes certain preparations to the spec before it's sent to be applied
func (FeatureManager) PrepareSpec(ctx context.Context, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	return spec, nil
}

func (m FeatureManager) getComponentOutput(
	ctx context.Context,
	clusterID uint,
	helper outputHelper,
	endpoints []*pkgHelm.EndpointItem,
	pipelineSystemNamespace string,
	releaseName string,
	deploymentValues map[string]interface{},
) map[string]interface{} {
	var out = make(map[string]interface{})

	o := outputManager{
		outputHelper: helper,
		secretStore:  m.secretStore,
		logger:       m.logger,
	}

	writeSecretID(ctx, o, clusterID, out)
	writeUrl(o, endpoints, releaseName, out)
	writeVersion(o, deploymentValues, out)
	if err := writeServiceUrl(o, m.endpointsService, pipelineSystemNamespace, out); err != nil {
		m.logger.Warn(fmt.Sprintf("failed to get service url: %s", err.Error()))
	}

	return out
}
