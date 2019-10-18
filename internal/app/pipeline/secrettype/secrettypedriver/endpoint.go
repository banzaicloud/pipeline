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

package secrettypedriver

import (
	"context"

	"emperror.dev/errors"
	"github.com/go-kit/kit/endpoint"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/secrettype"
)

// MakeListSecretTypesEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListSecretTypesEndpoint(service secrettype.TypeService) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, _ interface{}) (interface{}, error) {
		return service.ListSecretTypes(ctx)
	})
}

type getSecretTypeRequest struct {
	SecretType string
}

type getSecretTypeError struct {
	err error
}

func (f getSecretTypeError) Failed() error {
	return f.err
}

// MakeGetSecretTypeEndpoint returns an endpoint for the matching method of the underlying service.
func MakeGetSecretTypeEndpoint(service secrettype.TypeService) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		r := req.(getSecretTypeRequest)

		secretType, err := service.GetSecretType(ctx, r.SecretType)
		if err != nil && errors.Is(err, secrettype.ErrNotSupportedSecretType) {
			return getSecretTypeError{err}, nil
		}

		return secretType, err
	})
}
