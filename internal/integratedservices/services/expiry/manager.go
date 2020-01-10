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

package expiry

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type expiryServiceManager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer
}

func (e expiryServiceManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	panic("implement me")
}

func (e expiryServiceManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	expirySpec := ServiceSpec{}
	if err := services.BindIntegratedServiceSpec(spec, &expirySpec); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: ExpiryIntegrationServiceName,
			Problem:               "failed to bind the expiry service specification",
		}
	}

	if err := expirySpec.Validate(); err != nil {
		return err
	}

	return nil
}

func (e expiryServiceManager) Name() string {
	return ExpiryIntegrationServiceName
}

func NewExpiryServiceManager() expiryServiceManager {
	return expiryServiceManager{}
}
