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

package workflow

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/goph/emperror"
)

type AzureClientFactory struct {
	secretStore pkeworkflow.SecretStore
}

func NewAzureClientFactory(secretStore pkeworkflow.SecretStore) *AzureClientFactory {
	return &AzureClientFactory{secretStore: secretStore}
}

func (f *AzureClientFactory) New(organizationID uint, secretID string) (*pkgAzure.CloudConnection, error) {
	s, err := f.secretStore.GetSecret(organizationID, secretID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get secret")
	}

	err = s.ValidateSecretType(pkgAzure.Provider)
	if err != nil {
		return nil, err
	}

	cc, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, pkgAzure.NewCredentials(s.GetValues()))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create cloud connection")
	}

	return cc, nil
}

func getOwnedTag(clusterName string) (string, string) {
	return fmt.Sprintf("kubernetesCluster-%s", clusterName), "owned"
}

func HasOwnedTag(clusterName string, tags map[string]string) bool {
	ownedTag := fmt.Sprintf("kubernetesCluster-%s", clusterName)

	v, ok := tags[ownedTag]

	return ok && v == "owned"
}

// func getSharedTag(clusterName string) (string, string) {
// 	return fmt.Sprintf("kubernetesCluster-%s", clusterName), "shared"
// }

func tagsFrom(key, value string) map[string]string {
	return map[string]string{
		key: value,
	}
}
