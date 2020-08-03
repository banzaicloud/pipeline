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
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

// IntegratedServiceManager implements the Monitoring integrated service manager
type IntegratedServiceManager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer

	clusterGetter    integratedserviceadapter.ClusterGetter
	secretStore      services.SecretStore
	endpointsService endpoints.EndpointService
	helmService      services.HelmService
	config           Config
	logger           common.Logger
}

func MakeIntegratedServiceManager(
	clusterGetter integratedserviceadapter.ClusterGetter,
	secretStore services.SecretStore,
	endpointsService endpoints.EndpointService,
	helmService services.HelmService,
	config Config,
	logger common.Logger,
) IntegratedServiceManager {
	return IntegratedServiceManager{
		clusterGetter:    clusterGetter,
		secretStore:      secretStore,
		endpointsService: endpointsService,
		helmService:      helmService,
		config:           config,
		logger:           logger,
	}
}

// Name returns the integrated service' name
func (IntegratedServiceManager) Name() string {
	return integratedServiceName
}

// GetOutput returns the Monitoring integrated service'output
func (m IntegratedServiceManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return nil, integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
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

	endpoints, err := m.endpointsService.List(ctx, kubeConfig, prometheusOperatorReleaseName)
	if err != nil {
		m.logger.Warn(fmt.Sprintf("failed to list endpoints: %s", err.Error()))
	}

	operatorValues := m.config.Charts.Operator.Values
	pushgatewayValues := m.config.Charts.Pushgateway.Values

	out := integratedservices.IntegratedServiceOutput{
		"grafana":      m.getComponentOutput(ctx, clusterID, newGrafanaOutputHelper(kubeConfig, boundSpec), endpoints, m.config.Namespace, prometheusOperatorReleaseName, operatorValues, m.config.Images.Grafana),
		"prometheus":   m.getComponentOutput(ctx, clusterID, newPrometheusOutputHelper(kubeConfig, boundSpec), endpoints, m.config.Namespace, prometheusOperatorReleaseName, operatorValues, m.config.Images.Prometheus),
		"alertmanager": m.getComponentOutput(ctx, clusterID, newAlertmanagerOutputHelper(kubeConfig, boundSpec), endpoints, m.config.Namespace, prometheusOperatorReleaseName, operatorValues, m.config.Images.Alertmanager),
		"pushgateway":  m.getComponentOutput(ctx, clusterID, newPushgatewayOutputHelper(kubeConfig, boundSpec), endpoints, m.config.Namespace, prometheusPushgatewayReleaseName, pushgatewayValues, m.config.Images.Pushgateway),
		"prometheusOperator": map[string]interface{}{
			"version": m.config.Charts.Operator.Version,
		},
	}

	return out, nil
}

// ValidateSpec validates a Monitoring integrated service specification
func (IntegratedServiceManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
		}
	}

	if err := boundSpec.Validate(); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
		}
	}

	return nil
}

func (m IntegratedServiceManager) getComponentOutput(
	ctx context.Context,
	clusterID uint,
	helper outputHelper,
	endpoints []*pkgHelm.EndpointItem,
	pipelineSystemNamespace string,
	releaseName string,
	values map[string]interface{},
	config ImageConfig,
) map[string]interface{} {
	out := make(map[string]interface{})

	o := outputManager{
		outputHelper: helper,
		secretStore:  m.secretStore,
		logger:       m.logger,
	}

	writeSecretID(ctx, o, clusterID, out)
	writeURL(o, endpoints, releaseName, out)
	// TODO (colin): put back after the values can came from config
	// writeVersion(o, values, out)
	out[versionKey] = config.Tag
	if err := writeServiceURL(ctx, o, m.endpointsService, pipelineSystemNamespace, out); err != nil {
		m.logger.Warn(fmt.Sprintf("failed to get service url: %s", err.Error()))
	}

	return out
}
