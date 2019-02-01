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

package azure

import (
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/providers/azure"

	azureObjectstore "github.com/banzaicloud/pipeline/pkg/providers/azure/objectstore"
	"github.com/goph/emperror"

	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

// Secret describes values for Azure access
type Secret struct {
	// For general access
	ClientID       string `json:"AZURE_CLIENT_ID,omitempty"`
	ClientSecret   string `json:"AZURE_CLIENT_SECRET,omitempty"`
	SubscriptionID string `json:"AZURE_SUBSCRIPTION_ID,omitempty"`
	TenantID       string `json:"AZURE_TENANT_ID,omitempty"`

	// For bucket access
	ResourceGroup  string `json:"AZURE_RESOURCE_GROUP,omitempty"`
	StorageAccount string `json:"AZURE_STORAGE_ACCOUNT_ID,omitempty"`
	StorageKey     string `json:"AZURE_STORAGE_KEY,omitempty"`
}

// GetSecretForBucket gets formatted secret for ARK backup bucket
func GetSecretForBucket(secret *secret.SecretItemResponse, storageAccount string, resourceGroup string) (Secret, error) {
	s := getSecret(secret)
	s.StorageAccount = storageAccount
	s.ResourceGroup = resourceGroup

	storageAccountClient, err := azureObjectstore.NewAuthorizedStorageAccountClientFromSecret(*azure.NewCredentials(secret.Values))
	if err != nil {
		return Secret{}, emperror.Wrap(err, "failed to create storage account client")
	}

	key, err := storageAccountClient.GetStorageAccountKey(resourceGroup, storageAccount)
	if err != nil {
		return Secret{}, err
	}

	s.StorageKey = key

	return s, nil
}

// GetSecretForCluster gets formatted secret for cluster
func GetSecretForCluster(secret *secret.SecretItemResponse, clusterName, location, resourceGroup string) (Secret, error) {
	s := getSecret(secret)
	s.ResourceGroup = fmt.Sprintf("MC_%s_%s_%s", resourceGroup, clusterName, location)

	return s, nil
}

// getSecret gets formatted secret for ARK
func getSecret(secret *secret.SecretItemResponse) Secret {
	return Secret{
		ClientID:       secret.Values[pkgSecret.AzureClientID],
		ClientSecret:   secret.Values[pkgSecret.AzureClientSecret],
		SubscriptionID: secret.Values[pkgSecret.AzureSubscriptionID],
		TenantID:       secret.Values[pkgSecret.AzureTenantID],
	}
}
