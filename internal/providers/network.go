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
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/network"
	"github.com/banzaicloud/pipeline/internal/providers/alibaba"
	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/internal/providers/azure"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	"github.com/banzaicloud/pipeline/internal/providers/oracle"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
)

// ServiceParams describes all parameters necessary to create cloud provider agnostic VPC network queries
type ServiceParams struct {
	Logger            logrus.FieldLogger
	Provider          string
	Region            string
	ResourceGroupName string
	Secret            *secret.SecretItemResponse
}

// NewNetworkService returns a new network Service instance of the specified provider
func NewNetworkService(params ServiceParams) (network.Service, error) {
	switch params.Provider {
	case providers.Alibaba:
		return alibaba.NewNetworkService(params.Region, params.Secret, params.Logger)
	case providers.Amazon:
		return amazon.NewNetworkService(params.Region, params.Secret, params.Logger)
	case providers.Azure:
		return azure.NewNetworkService(params.ResourceGroupName, params.Secret, params.Logger)
	case providers.Google:
		return google.NewNetworkService(params.Region, params.Secret, params.Logger)
	case providers.Oracle:
		return oracle.NewNetworkService(params.Region, params.Secret, params.Logger)
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}
