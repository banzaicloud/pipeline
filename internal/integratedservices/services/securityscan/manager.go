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

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type IntegratedServiceManager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer

	config Config
	logger common.Logger
}

// Name returns the name of the integrated service
func (f IntegratedServiceManager) Name() string {
	return IntegratedServiceName
}

// MakeIntegratedServiceManager creates asecurity scan integrated service manager instance
func MakeIntegratedServiceManager(logger common.Logger, config Config) IntegratedServiceManager {
	return IntegratedServiceManager{
		config: config,
		logger: logger,
	}
}

func (f IntegratedServiceManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	securityScanSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: IntegratedServiceName,
			Problem:               err.Error(),
		}
	}

	if err := securityScanSpec.Validate(f.config.PipelineNamespace); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: IntegratedServiceName,
			Problem:               err.Error(),
		}
	}

	return nil
}

func (f IntegratedServiceManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	// todo read these through the helm service?
	out := map[string]interface{}{
		"anchore": map[string]interface{}{
			// leave this for backwards compatibility
			// to be populated (externally) via direct call to the configured anchore service
			"version": "",
		},
		"imageValidator": map[string]interface{}{
			"version": f.config.Webhook.Version,
		},
	}

	return out, nil
}
