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

	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

const (
	urlKey      = "url"
	secretIDKey = "secretId"
	versionKey  = "version"
)

type baseOutput struct {
	ingress  ingressSpec
	secretID string
	enabled  bool
}

func (o baseOutput) getSecretID() string {
	return o.secretID
}

func (o baseOutput) isEnabled() bool {
	return o.enabled
}

func (o baseOutput) getIngress() ingressSpec {
	return o.ingress
}

type outputHelper interface {
	getOutputType() string
	getDeploymentValueParentKey() string
	getTopLevelDeploymentKey() string
	getGeneratedSecretName(clusterID uint) string
	getIngress() ingressSpec
	isEnabled() bool
	getSecretID() string
}

type outputManager struct {
	outputHelper
	secretStore features.SecretStore
	logger      common.Logger
}

func writeVersion(m outputManager, deploymentValues map[string]interface{}, output map[string]interface{}) {
	if m.isEnabled() && deploymentValues != nil {
		var ok = true
		if m.getTopLevelDeploymentKey() != "" {
			deploymentValues, ok = deploymentValues[m.getTopLevelDeploymentKey()].(map[string]interface{})
		}
		if ok {
			output[versionKey] = m.getVersionFromValues(deploymentValues)
		}
	}
}

func writeUrl(m outputManager, endpoints []*pkgHelm.EndpointItem, output map[string]interface{}) {
	if m.isEnabled() {
		ingress := m.getIngress()
		if ingress.Enabled && endpoints != nil {
			output[urlKey] = getEndpointUrl(endpoints, ingress.Path)
		}
	}
}

func writeSecretID(ctx context.Context, m outputManager, clusterID uint, output map[string]interface{}) {
	if m.isEnabled() {
		var generatedSecretName = m.getGeneratedSecretName(clusterID)
		if m.getSecretID() == "" && generatedSecretName != "" {
			secretID, err := m.secretStore.GetIDByName(ctx, generatedSecretName)
			if err != nil {
				m.logger.Warn(fmt.Sprintf("failed to get generated %s secret", m.getOutputType()))
				return
			}

			output[secretIDKey] = secretID
		}
	}
}

func (m *outputManager) getVersionFromValues(values map[string]interface{}) string {
	if specValues, ok := values[m.getDeploymentValueParentKey()].(map[string]interface{}); ok {
		if image, ok := specValues["image"].(map[string]interface{}); ok {
			return image["tag"].(string)
		}
	}
	return ""
}

func getEndpointUrl(endpoints []*pkgHelm.EndpointItem, path string) string {
	for _, ep := range endpoints {
		for _, url := range ep.EndPointURLs {
			if url.Path == path {
				return url.URL
			}
		}
	}
	return ""
}
