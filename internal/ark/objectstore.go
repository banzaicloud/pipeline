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

package ark

import (
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"

	"github.com/banzaicloud/pipeline/internal/ark/providers/amazon"
	"github.com/banzaicloud/pipeline/internal/ark/providers/azure"
	"github.com/banzaicloud/pipeline/internal/ark/providers/google"
	iProviders "github.com/banzaicloud/pipeline/internal/providers"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
)

type objectStoreGetter struct {
	objectStore velero.ObjectStore
}

func (o *objectStoreGetter) GetObjectStore(provider string) (velero.ObjectStore, error) {
	return o.objectStore, nil
}

func NewObjectStoreGetter(ctx iProviders.ObjectStoreContext) (*objectStoreGetter, error) {
	store, err := NewObjectStore(ctx)
	if err != nil {
		return nil, err
	}
	return &objectStoreGetter{
		store,
	}, nil
}

// NewObjectStore gets a initialized ObjectStore for the given provider
func NewObjectStore(ctx iProviders.ObjectStoreContext) (velero.ObjectStore, error) {
	switch ctx.Provider {
	case providers.Google:
		return google.NewObjectStore(ctx)
	case providers.Amazon:
		return amazon.NewObjectStore(ctx)
	case providers.Azure:
		return azure.NewObjectStore(ctx)
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}
