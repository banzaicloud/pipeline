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

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type expiryServiceOperator struct {
	expiryService  ExpiryService
	specBinderFunc binderFunc
	logger         common.Logger
}

func NewExpiryServiceOperator(expiryService ExpiryService, binderFn binderFunc, logger common.Logger) expiryServiceOperator {
	return expiryServiceOperator{
		expiryService:  expiryService,
		specBinderFunc: binderFn,
		logger:         logger,
	}
}

func (e expiryServiceOperator) Name() string {
	return ServiceName
}

func (e expiryServiceOperator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	expirySpec := ServiceSpec{}
	if err := e.specBinderFunc(spec, &expirySpec); err != nil {
		return errors.WrapIf(err, "failed to bind the expiry service specification")
	}

	if err := e.expiryService.Expire(ctx, clusterID, expirySpec.Date); err != nil {
		return errors.WrapIf(err, "failed to expire the resource")
	}

	return nil
}

func (e expiryServiceOperator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	if err := e.expiryService.CancelExpiry(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to cancel the expiry")
	}

	return nil
}
