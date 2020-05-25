// Copyright © 2020 Banzai Cloud
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

package ingress

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type Manager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer

	config      Config
	helmService services.HelmService
	logger      services.Logger
}

func NewManager(config Config, helmService services.HelmService, logger services.Logger) Manager {
	return Manager{
		config:      config,
		helmService: helmService,
		logger:      logger,
	}
}

// Name returns the integrated service's name.
func (Manager) Name() string {
	return ServiceName
}

func (m Manager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	var output integratedservices.IntegratedServiceOutput

	var boundSpec Spec
	if err := services.BindIntegratedServiceSpec(spec, &boundSpec); err != nil {
		return nil, errors.WrapIf(err, "failed to bind spec")
	}

	switch boundSpec.Controller.Type {
	case ControllerTraefik:
		traefikOutput := make(map[string]interface{})

		rel, err := m.helmService.GetDeployment(ctx, clusterID, m.config.ReleaseName, m.config.Namespace)
		if err != nil {
			m.logger.Warn(err.Error(), map[string]interface{}{
				"clusterId":   clusterID,
				"releaseName": m.config.ReleaseName,
			})
		}

		if rel != nil {
			traefikOutput["version"] = rel.ChartVersion
		} else {
			traefikOutput["version"] = m.config.Charts.Traefik.Version
		}

		output = set(output, "traefik", traefikOutput)
	}

	return output, nil
}

func (m Manager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	var boundSpec Spec
	if err := services.BindIntegratedServiceSpec(spec, &boundSpec); err != nil {
		return errors.WrapIf(err, "failed to bind spec")
	}

	return boundSpec.Validate(m.config)
}

func set(dst map[string]interface{}, key string, val interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	dst[key] = val
	return dst
}
