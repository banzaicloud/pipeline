// Copyright © 2021 Banzai Cloud
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

package backup

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/backup"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type SpecConverter struct {
}

func (s SpecConverter) ConvertSpec(ctx context.Context, service v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceSpec, error) {
	if service.Spec.Backup.Spec == nil {
		return integratedservices.IntegratedServiceSpec{}, nil
	}
	return convert(service.Spec.Backup.Spec)
}

func convert(spec *backup.ServiceSpec) (integratedservices.IntegratedServiceSpec, error) {
	var decoded integratedservices.IntegratedServiceSpec
	if err := mapstructure.Decode(spec, &decoded); err != nil {
		return decoded, errors.WrapIf(err, "failed to convert typed integrated service spec")
	}
	return decoded, nil
}
