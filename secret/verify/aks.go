// Copyright © 2018 Banzai Cloud
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

package verify

import (
	"github.com/banzaicloud/azure-aks-client/client"
	"github.com/banzaicloud/azure-aks-client/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

// aksVerify for validation AKS credentials
type aksVerify struct {
	credential *cluster.AKSCredential
}

// CreateAKSSecret create a new 'aksVerify' instance
func CreateAKSSecret(values map[string]string) *aksVerify {
	return &aksVerify{
		credential: CreateAKSCredentials(values),
	}
}

// VerifySecret validates AKS credentials
func (a *aksVerify) VerifySecret() (err error) {
	manager, err := client.GetAKSClient(a.credential)
	if err != nil {
		return
	}

	return client.ValidateCredentials(manager)
}

// CreateAKSCredentials create an 'AKSCredential' instance from secret's values
func CreateAKSCredentials(values map[string]string) *cluster.AKSCredential {
	return &cluster.AKSCredential{
		ClientId:       values[pkgSecret.AzureClientId],
		ClientSecret:   values[pkgSecret.AzureClientSecret],
		SubscriptionId: values[pkgSecret.AzureSubscriptionId],
		TenantId:       values[pkgSecret.AzureTenantId],
	}
}
