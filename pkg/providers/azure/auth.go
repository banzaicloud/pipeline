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

package azure

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/banzaicloud/pipeline/pkg/secret"
)

// ServicePrincipal represents Azure service principal data
type ServicePrincipal struct {
	ClientID     string
	ClientSecret string
	TenantID     string
}

// Credentials represents Azure credential data
type Credentials struct {
	ServicePrincipal
	SubscriptionID string
}

// NewCredentials creates a new Credentials instance from a secret's values
func NewCredentials(values map[string]string) *Credentials {
	return &Credentials{
		ServicePrincipal: ServicePrincipal{
			ClientID:     values[secret.AzureClientID],
			ClientSecret: values[secret.AzureClientSecret],
			TenantID:     values[secret.AzureTenantID],
		},
		SubscriptionID: values[secret.AzureSubscriptionID],
	}
}

// GetAuthorizer returns autorest Authorizer with the specified service principal in the specified environment
func GetAuthorizer(sp *ServicePrincipal, env *azure.Environment) (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, sp.TenantID)
	if err != nil {
		return nil, err
	}
	token, err := adal.NewServicePrincipalToken(*oauthConfig, sp.ClientID, sp.ClientSecret, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(token), nil
}
