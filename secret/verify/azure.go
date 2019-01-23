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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
)

// AzureSecretVerifier represents a secret verifier for Azure secrets
type AzureSecretVerifier struct {
	*pkgAzure.Credentials
}

// CreateAzureSecretVerifier creates a new Azure secret verifier
func CreateAzureSecretVerifier(values map[string]string) AzureSecretVerifier {
	return AzureSecretVerifier{pkgAzure.NewCredentials(values)}
}

// VerifySecret validates Azure credentials
func (a AzureSecretVerifier) VerifySecret() error {
	cc, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, a.Credentials)
	if err != nil {
		return err
	}
	missing, err := checkRequiredProviders(cc.GetProvidersClient())
	if err != nil {
		return err
	}
	if len(missing) != 0 {
		return fmt.Errorf("required providers are not registered: %s", strings.Join(missing, ", "))
	}
	return nil
}

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
