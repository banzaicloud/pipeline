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

// ObjectStore is the interface that cloud specific object store implementation must implement.
type ObjectStore interface {
	// CreateBucket creates a new bucket in the object store.
	CreateBucket(string) error

	// ListBuckets lists the current buckets in the object store.
	ListBuckets() ([]string, error)

	// CheckBucket checks the status of the given bucket.
	CheckBucket(string) error

	// DeleteBucket removes a bucket from the object store.
	DeleteBucket(string) error
}
