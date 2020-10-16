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

package providers

import (
	"time"

	"github.com/banzaicloud/pipeline/pkg/objectstore"
)

// ProviderObjectStore is the interface that cloud specific object store implementation must implement.
type ProviderObjectStore interface {
	objectstore.ObjectStore

	// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter and
	// starts with a specified prefix.
	ListObjectKeyPrefixesStartingWithPrefix(bucketName string, prefix string, delimeter string) ([]string, error)
}

type ObjectStore struct {
	ProviderObjectStore
}

// This actually does nothing in this implementation
func (o *ObjectStore) Init(config map[string]string) error {
	return nil
}

// CreateSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	return o.GetSignedURL(bucket, key, ttl)
}

// ListObjects gets all keys with the given prefix from the bucket
func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	return o.ListObjectsWithPrefix(bucket, prefix)
}

// ListCommonPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	return o.ListObjectKeyPrefixesStartingWithPrefix(bucket, prefix, delimiter)
}

// ObjectExists checks if there is an object with the given key in the object storage bucket.
func (o *ObjectStore) ObjectExists(bucket, key string) (bool, error) {
	_, err := o.GetObject(bucket, key)
	if objectstore.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return true, err
	}
	return true, nil
}
