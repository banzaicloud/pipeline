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

// NetworkContext describes all parameters necessary to create cloud provider agnostic VPC network queries
type NetworkContext struct {
	Logger   logrus.FieldLogger
	Provider string
	Secret   *secret.SecretItemResponse
}

// ListNetworks returns the VPC networks of the organization at the specified provider
func ListNetworks(networkCtx NetworkContext) ([]network.Network, error) {
	switch networkCtx.Provider {
	case providers.Google:
		return google.ListNetworks(networkCtx.Secret, networkCtx.Logger)
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}

// ListSubnets returns the VPC subnetworks of the organization in the specified VPC network at the specified provider
func ListSubnets(networkCtx NetworkContext, networkID string) ([]network.Subnet, error) {
	switch networkCtx.Provider {
	case providers.Google:
		return google.ListSubnets(networkCtx.Secret, networkID, networkCtx.Logger)
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}

// ListRouteTables returns the VPC route tables of the organization in the specified VPC network at the specified provider
func ListRouteTables(networkCtx NetworkContext, networkID string) ([]network.RouteTable, error) {
	switch networkCtx.Provider {
	case providers.Google:
		return google.ListRouteTables(networkCtx.Secret, networkID, networkCtx.Logger)
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}
