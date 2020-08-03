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

package logging

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

// IntegratedServiceManager implements the Logging integrated service manager
type IntegratedServicesManager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer

	clusterGetter    integratedserviceadapter.ClusterGetter
	secretStore      services.SecretStore
	endpointsService endpoints.EndpointService
	config           Config
	logger           common.Logger
}

func MakeIntegratedServiceManager(
	clusterGetter integratedserviceadapter.ClusterGetter,
	secretStore services.SecretStore,
	endpointsService endpoints.EndpointService,
	config Config,
	logger common.Logger,
) IntegratedServicesManager {
	return IntegratedServicesManager{
		clusterGetter:    clusterGetter,
		secretStore:      secretStore,
		endpointsService: endpointsService,
		config:           config,
		logger:           logger,
	}
}

// Name returns the integrated service' name
func (IntegratedServicesManager) Name() string {
	return integratedServiceName
}

func (m IntegratedServicesManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
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

	endpoints, err := m.endpointsService.List(ctx, kubeConfig, lokiReleaseName)
	if err != nil {
		m.logger.Warn(fmt.Sprintf("failed to list endpoints: %s", err.Error()))
	}

	return integratedservices.IntegratedServiceOutput{
		"logging": map[string]interface{}{
			"operatorVersion":  m.config.Charts.Operator.Version,
			"fluentdVersion":   m.config.Images.Fluentd.Tag,
			"fluentbitVersion": m.config.Images.Fluentbit.Tag,
		},
		"loki": m.getLokiOutput(ctx, boundSpec, endpoints, kubeConfig, clusterID),
	}, nil
}

func (m IntegratedServicesManager) getLokiOutput(
	ctx context.Context,
	spec integratedServiceSpec,
	endpoints []*pkgHelm.EndpointItem,
	kubeConfig []byte,
	clusterID uint,
) map[string]interface{} {
	if spec.Loki.Enabled {
		serviceUrl, err := getLokiServiceURL(ctx, spec.Loki, kubeConfig, m.endpointsService, m.config.Namespace)
		if err != nil {
			m.logger.Warn("failed to get Loki service url")
		}
		return map[string]interface{}{
			"url":        getLokiEndpoint(endpoints, spec.Loki),
			"version":    m.config.Images.Loki.Tag,
			"serviceUrl": serviceUrl,
			"secretId":   m.getLokiSecretID(ctx, spec.Loki, clusterID),
		}
	}
	return nil
}

func getLokiEndpoint(endpoints []*pkgHelm.EndpointItem, spec lokiSpec) string {
	if spec.Ingress.Enabled && endpoints != nil {
		return getEndpointUrl(endpoints, spec.Ingress.Path, lokiReleaseName)
	}
	return ""
}

func getEndpointUrl(endpoints []*pkgHelm.EndpointItem, path, releaseName string) string {
	for _, ep := range endpoints {
		for _, url := range ep.EndPointURLs {
			if url.Path == path && url.ReleaseName == releaseName {
				return url.URL
			}
		}
	}
	return ""
}

func getLokiServiceURL(
	ctx context.Context,
	spec lokiSpec,
	k8sConfig []byte,
	service endpoints.EndpointService,
	pipelineSystemNamespace string,
) (string, error) {
	if spec.Enabled {
		url, err := service.GetServiceURL(ctx, k8sConfig, lokiServiceName, pipelineSystemNamespace)
		if err != nil {
			return "", errors.WrapIf(err, "failed to get service")
		}
		return url, nil
	}

	return "", nil
}

func (m IntegratedServicesManager) getLokiSecretID(ctx context.Context, spec lokiSpec, clusterID uint) string {
	if spec.Enabled && spec.Ingress.Enabled {
		generatedSecretName := getLokiSecretName(clusterID)
		if spec.Ingress.SecretID == "" && generatedSecretName != "" {
			secretID, err := m.secretStore.GetIDByName(ctx, generatedSecretName)
			if err != nil {
				m.logger.Warn("failed to get generated Loki secret")
				return ""
			}

			return secretID
		}
	}
	return ""
}

func (IntegratedServicesManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	vaultSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return err
	}

	if err := vaultSpec.Validate(); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
		}
	}

	return nil
}

func (IntegratedServicesManager) PrepareSpec(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceSpec, error) {
	return spec, nil
}
