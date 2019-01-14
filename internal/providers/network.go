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

package providers

import (
	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
)

// ServiceParams describes all parameters necessary to create cloud provider agnostic VPC network queries
type ServiceParams struct {
	Logger   logrus.FieldLogger
	Provider string
	Secret   *secret.SecretItemResponse
}

// NewNetworkService returns a new network Service instance of the specified provider
func NewNetworkService(params ServiceParams) (network.Service, error) {
	switch params.Provider {
	case providers.Google:
		return google.NewNetworkService(params.Secret, params.Logger)
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}
