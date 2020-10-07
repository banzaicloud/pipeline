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

package types

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/banzaicloud/pipeline/internal/secret"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
)

const Azure = "azure"

const (
	FieldAzureClientID       = "AZURE_CLIENT_ID"
	FieldAzureClientSecret   = "AZURE_CLIENT_SECRET"
	FieldAzureTenantID       = "AZURE_TENANT_ID"
	FieldAzureSubscriptionID = "AZURE_SUBSCRIPTION_ID"
)

type AzureType struct{}

func (AzureType) Name() string {
	return Azure
}

func (AzureType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldAzureClientID, Required: true, IsSafeToDisplay: true, Description: "Your application client id"},
			{Name: FieldAzureClientSecret, Required: true, Description: "Your client secret id"},
			{Name: FieldAzureTenantID, Required: true, IsSafeToDisplay: true, Description: "Your tenant id"},
			{Name: FieldAzureSubscriptionID, Required: true, IsSafeToDisplay: true, Description: "Your subscription id"},
		},
	}
}

func (t AzureType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}

// TODO: rewrite this function!
func (AzureType) Verify(data map[string]string) error {
	creds := pkgAzure.NewCredentials(data)

	cloudConnection, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, creds)
	if err != nil {
		return err
	}

	missing, err := checkRequiredProviders(cloudConnection.GetProvidersClient())
	if err != nil {
		return secret.NewValidationError(err.Error(), nil)
	}

	if len(missing) != 0 {
		return secret.NewValidationError(fmt.Sprintf("required providers are not registered: %s", strings.Join(missing, ", ")), nil)
	}

	return nil
}

// TODO: rewrite this functions
func checkRequiredProviders(client *pkgAzure.ProvidersClient) (missing []string, err error) {
	providersToCheck := map[string]bool{
		"Microsoft.Compute":          true,
		"Microsoft.ContainerService": true,
		"Microsoft.Network":          true,
		"Microsoft.Storage":          true,
	}

	rp, err := client.List(context.Background(), nil, "")
	if err != nil {
		return
	}

	for len(providersToCheck) != 0 && rp.NotDone() {
		for _, provider := range rp.Values() {
			namespace := *provider.Namespace
			if providersToCheck[namespace] {
				if "Registered" != *provider.RegistrationState {
					missing = append(missing, namespace)
				}

				delete(providersToCheck, namespace)
			}

			if len(providersToCheck) == 0 {
				return
			}
		}

		if err = rp.NextWithContext(context.Background()); err != nil {
			return
		}
	}

	if len(providersToCheck) != 0 {
		notFound := make([]string, 0, len(providersToCheck))

		for p := range providersToCheck {
			notFound = append(notFound, p)
		}

		errorMsg := fmt.Sprintf("required providers missing from providers listing: %s", strings.Join(notFound, ", "))

		err = errors.New(errorMsg)
	}

	return
}
