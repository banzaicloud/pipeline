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

package dns

import (
	"context"

	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type OutputResolver struct{}

func (o OutputResolver) Resolve(ctx context.Context, instance v1alpha1.ServiceInstance) (integratedservices.IntegratedServiceOutput, error) {
	return integratedservices.IntegratedServiceOutput{
		"externalDns": map[string]string{
			"version": instance.Status.Version,
		},
	}, nil
}
