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

	logger common.Logger
}

// Name returns the name of the integrated service
func (f IntegratedServiceManager) Name() string {
	return IntegratedServiceName
}

//MakeIntegratedServiceManager creates asecurity scan integrated service manager instance
func MakeIntegratedServiceManager(logger common.Logger) IntegratedServiceManager {
	return IntegratedServiceManager{
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

	if err := securityScanSpec.Validate(); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: IntegratedServiceName,
			Problem:               err.Error(),
		}
	}

	return nil
}

func (f IntegratedServiceManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	out := map[string]interface{}{
		// todo add "real" anchore version
		"anchore": map[string]interface{}{
			"version": securityScanChartVersion,
		},
		"imageValidator": map[string]interface{}{
			"version": imageValidatorVersion,
		},
	}

	return out, nil
}
