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
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// NewResourceGroupClientFromSecret returns an Azure resource groups client from a secret.
func NewResourceGroupClientFromSecret(secret map[string]string) resources.GroupsClient {
	return resources.NewGroupsClient(secret[pkgSecret.AzureSubscriptionId])
}

// NewAuthorizedResourceGroupClientFromSecret returns an authorized Azure resource groups client from a secret.
func NewAuthorizedResourceGroupClientFromSecret(secret map[string]string) (resources.GroupsClient, error) {
	credentials := NewClientCredentialsConfigFromSecret(secret)

	client := NewResourceGroupClientFromSecret(secret)

	authorizer, err := credentials.Authorizer()
	if err != nil {
		return client, errors.Wrap(err, "failed to authorize")
	}

	client.Authorizer = authorizer

	return client, nil
}

// ResourceGroupClientFactory creates a new resource group client.
type ResourceGroupClientFactory struct {
	secrets secretClient
}

// NewResourceGroupClientFactory returns a new resource group client factory.
func NewResourceGroupClientFactory(secrets secretClient) *ResourceGroupClientFactory {
	return &ResourceGroupClientFactory{
		secrets: secrets,
	}
}

func (f *ResourceGroupClientFactory) New(organizationID uint, secretID string) (resources.GroupsClient, error) {
	var client resources.GroupsClient

	secret, err := f.secrets.Get(organizationID, secretID)
	if err != nil {
		return client, emperror.With(
			emperror.Wrap(err, "failed to get secret"),
			"organization-id", organizationID,
			"secret-id", secretID,
		)
	}

	return NewAuthorizedResourceGroupClientFromSecret(secret)
}

// CreateResourceGroup creates a resource group in an idempotent way.
func CreateResourceGroup(ctx context.Context, client resources.GroupsClient, resourceGroup string, location string) error {
	res, err := client.CheckExistence(ctx, resourceGroup)
	if err != nil {
		return emperror.With(
			errors.Wrap(err, "failed to check resource group existence"),
			"resource-group", resourceGroup,
		)
	}

	// Resource group does not exist; create it
	if res.StatusCode == http.StatusNotFound {
		result, err := client.CreateOrUpdate(
			ctx,
			resourceGroup,
			resources.Group{
				Location: to.StringPtr(location),
			},
		)
		if err != nil {
			return emperror.With(
				errors.Wrap(err, "failed to create resource group"),
				"resource-group", resourceGroup,
				"location", location,
			)
		}
		defer result.Body.Close()
	}

	return nil
}
