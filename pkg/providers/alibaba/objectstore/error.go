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

type errBucketAlreadyExists struct {
	bucketName string
}

func (e errBucketAlreadyExists) Error() string          { return "bucket already exists" }
func (e errBucketAlreadyExists) AlreadyExists() bool    { return true }
func (e errBucketAlreadyExists) Context() []interface{} { return []interface{}{"bucket", e.bucketName} }

type errBucketNotFound struct {
	bucketName string
}

func (e errBucketNotFound) Error() string          { return "bucket not found" }
func (e errBucketNotFound) NotFound() bool         { return true }
func (e errBucketNotFound) Context() []interface{} { return []interface{}{"bucket", e.bucketName} }

type errObjectNotFound struct {
	bucketName string
	objectName string
}

func (e errObjectNotFound) Error() string  { return "object not found" }
func (e errObjectNotFound) NotFound() bool { return true }
func (e errObjectNotFound) Context() []interface{} {
	return []interface{}{"bucket", e.bucketName, "object", e.objectName}
}
