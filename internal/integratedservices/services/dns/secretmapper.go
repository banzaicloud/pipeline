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

package dns

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/dns"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type SecretMapper struct {
	secretStore services.SecretStore
}

func NewSecretMapper(secretStore services.SecretStore) *SecretMapper {
	return &SecretMapper{
		secretStore: secretStore,
	}
}

func (s SecretMapper) ConvertSpec(ctx context.Context, service v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceSpec, error) {
	if service.Spec.DNS.Spec == nil {
		return integratedservices.IntegratedServiceSpec{}, nil
	}
	if services.IsManagedByPipeline(service.ObjectMeta) {
		secretID, err := s.secretStore.GetIDByName(ctx, service.Spec.DNS.Spec.ExternalDNS.Provider.SecretID)
		if err != nil {
			return integratedservices.IntegratedServiceSpec{}, errors.WrapIf(err, "unable to map dns secret name to secret id")
		}
		service.Spec.DNS.Spec.ExternalDNS.Provider.SecretID = secretID
	}
	return convert(service.Spec.DNS.Spec)
}

func convert(spec *dns.ServiceSpec) (integratedservices.IntegratedServiceSpec, error) {
	var decoded integratedservices.IntegratedServiceSpec
	if err := mapstructure.Decode(spec, &decoded); err != nil {
		return decoded, errors.WrapIf(err, "failed to convert typed integrated service spec")
	}
	return decoded, nil
}
