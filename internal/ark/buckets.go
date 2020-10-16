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
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/providers"
	pkgProviders "github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/auth"
)

// ValidateCreateBucketRequest validates a CreateBucketRequest
func ValidateCreateBucketRequest(req *api.CreateBucketRequest, org *auth.Organization) error {
	err := IsProviderSupported(req.Cloud)
	if err != nil {
		return errors.WrapIf(err, req.Cloud)
	}

	if req.Cloud == pkgProviders.Azure {
		if req.ResourceGroup == "" {
			return errors.New("error validating create bucket request: resourceGroup must not be empty")
		}
		if req.StorageAccount == "" {
			return errors.New("error validating create bucket request: storageAccount must not be empty")
		}
	}

	secret, err := GetSecretWithValidation(req.SecretID, org.ID, req.Cloud)
	if err != nil {
		return errors.WrapIf(err, "error validating create bucket request")
	}

	ctx := providers.ObjectStoreContext{
		Provider:       req.Cloud,
		Secret:         secret,
		Location:       req.Location,
		ResourceGroup:  req.ResourceGroup,
		StorageAccount: req.StorageAccount,
	}

	os, err := NewObjectStore(ctx)
	if err != nil {
		return errors.WrapIf(err, "error validating create bucket request")
	}

	// find only backups starting with the given prefix
	_, err = os.ListCommonPrefixes(req.BucketName, req.Prefix, "/")
	if err != nil {
		return errors.WrapIf(err, "error validating create bucket request")
	}

	return nil
}
