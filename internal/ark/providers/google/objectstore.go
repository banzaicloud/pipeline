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

package google

import (
	"time"

	"github.com/heptio/ark/pkg/cloudprovider"

	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	googleObjectstore "github.com/banzaicloud/pipeline/pkg/providers/google/objectstore"
)

type objectStore struct {
	objectstore.ObjectStore
}

// NewObjectStore creates a new objectStore
func NewObjectStore(ctx providers.ObjectStoreContext) (cloudprovider.ObjectStore, error) {

	config := googleObjectstore.Config{
		Region: ctx.Location,
	}

	credentials := googleObjectstore.Credentials{
		Type:                   ctx.Secret.Values[secrettype.Type],
		ProjectID:              ctx.Secret.Values[secrettype.ProjectId],
		PrivateKeyID:           ctx.Secret.Values[secrettype.PrivateKeyId],
		PrivateKey:             ctx.Secret.Values[secrettype.PrivateKey],
		ClientEmail:            ctx.Secret.Values[secrettype.ClientEmail],
		ClientID:               ctx.Secret.Values[secrettype.ClientId],
		AuthURI:                ctx.Secret.Values[secrettype.AuthUri],
		TokenURI:               ctx.Secret.Values[secrettype.TokenUri],
		AuthProviderX50CertURL: ctx.Secret.Values[secrettype.AuthX509Url],
		ClientX509CertURL:      ctx.Secret.Values[secrettype.ClientX509Url],
	}

	os, err := googleObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return &objectStore{
		ObjectStore: os,
	}, nil
}

// This actually does nothing in this implementation
func (o *objectStore) Init(config map[string]string) error {
	return nil
}

// CreateSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	return o.GetSignedURL(bucket, key, ttl)
}

// ListObjects gets all keys with the given prefix from the bucket
func (o *objectStore) ListObjects(bucket, prefix string) ([]string, error) {
	return o.ListObjectsWithPrefix(bucket, prefix)
}

// ListCommonPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *objectStore) ListCommonPrefixes(bucket, delimiter string) ([]string, error) {
	return o.ListObjectKeyPrefixes(bucket, delimiter)
}
