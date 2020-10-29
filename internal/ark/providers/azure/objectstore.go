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
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"

	arkProviders "github.com/banzaicloud/pipeline/internal/ark/providers"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	azureObjectstore "github.com/banzaicloud/pipeline/pkg/providers/azure/objectstore"
)

// NewObjectStore creates a new objectStore
func NewObjectStore(ctx providers.ObjectStoreContext) (velero.ObjectStore, error) {
	config := azureObjectstore.Config{
		StorageAccount: ctx.StorageAccount,
		ResourceGroup:  ctx.ResourceGroup,
	}

	return &arkProviders.ObjectStore{
		ProviderObjectStore: azureObjectstore.New(config, *azure.NewCredentials(ctx.Secret.Values)),
	}, nil
}
