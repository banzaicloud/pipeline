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

package objectstore

import (
	"io"
	"time"
)

// ObjectStore is the interface that cloud specific object store implementation must implement.
type ObjectStore interface {
	// CreateBucket creates a new bucket in the object store.
	CreateBucket(bucketName string) error

	// ListBuckets lists the current buckets in the object store.
	ListBuckets() ([]string, error)

	// CheckBucket checks the status of the given bucket.
	CheckBucket(bucketName string) error

	// DeleteBucket deletes a bucket from the object store.
	DeleteBucket(bucketName string) error

	// ListObjects gets all keys in the bucket.
	ListObjects(bucketName string) ([]string, error)

	// ListObjectsWithPrefix gets all keys with the given prefix from the bucket.
	ListObjectsWithPrefix(bucketName string, prefix string) ([]string, error)

	// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter.
	ListObjectKeyPrefixes(bucketName string, delimeter string) ([]string, error)

	// GetObject retrieves the object by it's key from the given bucket.
	GetObject(bucketName string, key string) (io.ReadCloser, error)

	// PutObject creates a new object using the data in body with the given key.
	PutObject(bucketName string, delimeter string, body io.Reader) error

	// DeleteObject deletes the object from the given bucket by it's key.
	DeleteObject(bucketName string, key string) error

	// GetSignedURL gives back a signed URL for the object that expires after the given ttl.
	GetSignedURL(bucketName string, key string, ttl time.Duration) (string, error)
}
