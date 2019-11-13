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

package logging

import (
	"emperror.dev/errors"

	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	azureObjectstore "github.com/banzaicloud/pipeline/pkg/providers/azure/objectstore"
)

type outputSecretInstallManagerAzure struct {
	baseOutputSecretInstallManager
}

func (m outputSecretInstallManagerAzure) generateSecretRequest(secretValues map[string]string, spec bucketSpec) (*pkgCluster.InstallSecretRequest, error) {

	credentials := *azure.NewCredentials(secretValues)

	storageAccountClient, err := azureObjectstore.NewAuthorizedStorageAccountClientFromSecret(credentials)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create storage account client")
	}
	sak, err := storageAccountClient.GetStorageAccountKey(spec.ResourceGroup, spec.StorageAccount)
	if err != nil {
		return nil, errors.WrapIf(err, "get storage account key failed")
	}

	return &pkgCluster.InstallSecretRequest{
		SourceSecretName: m.sourceSecretName,
		Namespace:        m.namespace,
		Spec: map[string]pkgCluster.InstallSecretRequestSpecItem{
			outputDefinitionSecretKeyAzureStorageAccount: {
				Value: spec.StorageAccount,
			},
			outputDefinitionSecretKeyAzureStorageAccess: {
				Value: sak,
			},
		},
		Update: true,
	}, nil
}
