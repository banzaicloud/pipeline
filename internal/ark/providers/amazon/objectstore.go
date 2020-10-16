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

package amazon

import (
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"

	arkProviders "github.com/banzaicloud/pipeline/internal/ark/providers"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	amazonObjectstore "github.com/banzaicloud/pipeline/pkg/providers/amazon/objectstore"
)

// NewObjectStore creates a new objectStore
func NewObjectStore(ctx providers.ObjectStoreContext) (velero.ObjectStore, error) {
	config := amazonObjectstore.Config{
		Region: ctx.Location,
	}

	credentials := amazonObjectstore.Credentials{
		AccessKeyID:     ctx.Secret.Values[secrettype.AwsAccessKeyId],
		SecretAccessKey: ctx.Secret.Values[secrettype.AwsSecretAccessKey],
	}

	os, err := amazonObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return &arkProviders.ObjectStore{
		ProviderObjectStore: os,
	}, nil
}
