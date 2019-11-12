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
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

// FeatureManager implements the Logging feature manager
type FeatureManager struct {
	clusterfeature.PassthroughFeatureSpecPreparer

	clusterGetter    clusterfeatureadapter.ClusterGetter
	secretStore      features.SecretStore
	endpointsService endpoints.EndpointService
	config           Config
	logger           common.Logger
}

func MakeFeatureManager(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	secretStore features.SecretStore,
	endpointsService endpoints.EndpointService,
	config Config,
	logger common.Logger,
) FeatureManager {
	return FeatureManager{
		clusterGetter:    clusterGetter,
		secretStore:      secretStore,
		endpointsService: endpointsService,
		config:           config,
		logger:           logger,
	}
}

// Name returns the feature's name
func (FeatureManager) Name() string {
	return featureName
}

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

	endpoints, err := m.endpointsService.List(kubeConfig, loggingOperatorReleaseName)
	if err != nil {
		m.logger.Warn(fmt.Sprintf("failed to list endpoints: %s", err.Error()))
	}

	return clusterfeature.FeatureOutput{
		"logging": map[string]interface{}{
			"operatorVersion": m.config.Charts.Operator.Version,
		},
		"loki": m.getLokiOutput(ctx, boundSpec, endpoints, kubeConfig, clusterID),
	}, nil
}

func (m FeatureManager) getLokiOutput(
	ctx context.Context,
	spec featureSpec,
	endpoints []*pkgHelm.EndpointItem,
	kubeConfig []byte,
	clusterID uint,
) map[string]interface{} {
	if spec.Loki.Enabled {
		serviceUrl, err := getLokiServiceURL(spec.Loki, kubeConfig, m.endpointsService, m.config.Namespace)
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
	spec lokiSpec,
	k8sConfig []byte,
	service endpoints.EndpointService,
	pipelineSystemNamespace string,
) (string, error) {
	if spec.Enabled {
		url, err := service.GetServiceURL(k8sConfig, lokiServiceName, pipelineSystemNamespace)
		if err != nil {
			return "", errors.WrapIf(err, "failed to get service")
		}
		return url, nil
	}

	return "", nil
}

func (m FeatureManager) getLokiSecretID(ctx context.Context, spec lokiSpec, clusterID uint) string {
	if spec.Enabled && spec.Ingress.Enabled {
		var generatedSecretName = getLokiSecretName(clusterID)
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

func (FeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	vaultSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return err
	}

	if err := vaultSpec.Validate(); err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	return nil
}

func (FeatureManager) PrepareSpec(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	return spec, nil
}
