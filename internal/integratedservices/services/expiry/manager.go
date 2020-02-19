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
)

type expiryServiceManager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer
	specBinderFunc binderFunc
}

func (e expiryServiceManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	return integratedservices.IntegratedServiceOutput{}, nil
}

func (e expiryServiceManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	var expirySpec ServiceSpec
	if err := e.specBinderFunc(spec, &expirySpec); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: ServiceName,
			Problem:               "failed to bind the expiry service specification",
		}
	}

	if err := expirySpec.Validate(); err != nil {
		return err
	}

	return nil
}

func (e expiryServiceManager) Name() string {
	return ServiceName
}

func NewExpiryServiceManager(specBinderFn binderFunc) expiryServiceManager {
	return expiryServiceManager{
		specBinderFunc: specBinderFn,
	}
}
