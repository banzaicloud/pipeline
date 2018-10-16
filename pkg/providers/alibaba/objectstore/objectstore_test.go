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
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
)

const bucketName = "banzaicloud-test-bucket"
const nonExistingBucketName = "a-asd8908sad-nonexisting-bucketname"
const existingBucket = "demo-bucket"

func getObjectStore(t *testing.T) *objectStore {
	t.Helper()

	if strings.TrimSpace(os.Getenv("TEST_OBJECTSTORE_INTEGRATION")) == "" {
		t.Skip("test disabled")
	}

	region := strings.TrimSpace(os.Getenv("ALIBABA_REGION_ID"))
	if region == "" {
		t.Skip("missing region")
	}

	return getObjectStoreWithRegion(t, region)
}

func getObjectStoreWithRegion(t *testing.T, region string) *objectStore {
	t.Helper()

	accessKeyID := strings.TrimSpace(os.Getenv("ALIBABA_ACCESS_KEY_ID"))
	accessKeySecret := strings.TrimSpace(os.Getenv("ALIBABA_ACCESS_KEY_SECRET"))

	if accessKeyID == "" || accessKeySecret == "" {
		t.Skip("missing necesseary env variables")
	}

	config := Config{
		Region: region,
	}

	credentials := Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: accessKeySecret,
	}

	ostore, err := New(config, credentials)
	if err != nil {
		t.Fatal("could not create object storage client: ", err.Error())
	}

	return ostore
}

func getBucketName(t *testing.T, bucketName string) string {
	t.Helper()

	prefix := strings.TrimSpace(os.Getenv("ALIBABA_BUCKET_PREFIX"))

	if prefix != "" {
		return fmt.Sprintf("%s-%s-%d", prefix, bucketName, time.Now().UnixNano())
	}

	return fmt.Sprintf("%s-%d", bucketName, time.Now().UnixNano())
}

func TestObjectStore_CreateAlreadyExistingBucket(t *testing.T) {
	var err error

	ostore := getObjectStore(t)

	err = ostore.CreateBucket(existingBucket)
	if _, ok := errors.Cause(err).(errBucketAlreadyExists); !ok {
		t.Error("error is not errBucketAlreadyExists: ", err.Error())
	}
}

func TestObjectStore_BucketNotFound(t *testing.T) {
	var err error

	ostore := getObjectStore(t)

	err = ostore.CheckBucket(nonExistingBucketName)
	if _, ok := errors.Cause(err).(errBucketNotFound); !ok {
		t.Fatal("error is not errBucketNotFound: ", err.Error())
	}
}

func TestObjectStore_ObjectNotFound(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	_, err = ostore.GetObject(bucketName, "test.txt")
	if _, ok := errors.Cause(err).(errObjectNotFound); !ok {
		t.Fatal("error is not errObjectNotFound: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_CreateDeleteBucket(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_ListBucket(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	buckets, err := ostore.ListBuckets()
	if err != nil {
		t.Error("could not list buckets: ", err.Error())
	}

	ok := false
	for _, name := range buckets {
		if name == bucketName {
			ok = true
		}
	}

	if !ok {
		t.Error("test bucket bucket not found")
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_CheckBucket(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.CheckBucket(bucketName)
	if err != nil {
		t.Error("bucket checking failed: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_CheckBucket_DiffRegion(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	diffRegion := strings.TrimSpace(os.Getenv("ALIBABA_DIFF_REGION_ID"))
	if diffRegion == "" {
		t.Skip("no different region was set")
	}
	if diffRegion == ostore.config.Region {
		t.Skip("same regions were set")
	}

	ostoreWithRegion := getObjectStoreWithRegion(t, diffRegion)
	err = ostoreWithRegion.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.CheckBucket(bucketName)
	if err != nil {
		t.Error("bucket checking failed: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_ListObjects(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	content := []byte("this a great content!")
	objectNames := []string{
		"test/test1.txt",
		"test/test2.txt",
		"demo/test1.txt",
		"demo2/test1.txt",
	}
	prefixedNames := []string{
		"test/test1.txt",
		"test/test2.txt",
	}
	prefixes := []string{
		"test",
		"demo",
		"demo2",
	}

	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	for _, objectName := range objectNames {
		err = ostore.PutObject(bucketName, objectName, bytes.NewReader(content))
		if err != nil {
			t.Error("could not create test object: ", err.Error())
		}
	}

	objects, err := ostore.ListObjects(bucketName)
	if err != nil {
		t.Error("could not list test objects: ", err.Error())
	}

	sort.Strings(objectNames)
	sort.Strings(objects)
	if !reflect.DeepEqual(objectNames, objects) {
		t.Error("retrieved objects differs")
	}

	objects, err = ostore.ListObjectsWithPrefix(bucketName, "test/")
	if err != nil {
		t.Error("could not list objects with prefix: ", err.Error())
	}

	sort.Strings(prefixedNames)
	sort.Strings(objects)
	if !reflect.DeepEqual(prefixedNames, objects) {
		t.Error("retrieved prefixed objects differs")
	}

	_prefixes, err := ostore.ListObjectKeyPrefixes(bucketName, "/")
	if err != nil {
		t.Error("could not list object key prefixes: ", err.Error())
	}

	sort.Strings(prefixes)
	sort.Strings(_prefixes)
	if !reflect.DeepEqual(prefixes, _prefixes) {
		t.Error("retrieved object key prefixes differs")
	}

	for _, objectName := range objectNames {
		err = ostore.DeleteObject(bucketName, objectName)
		if err != nil {
			t.Error("could not delete test object: ", err.Error())
		}
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_GetPutDeleteObject(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	objectName := "test/test.txt"
	content := []byte("this a great content!")

	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.PutObject(bucketName, objectName, bytes.NewReader(content))
	if err != nil {
		t.Error("could not create test object: ", err.Error())
	}

	reader, err := ostore.GetObject(bucketName, objectName)
	if err != nil {
		t.Error("could not get test object: ", err.Error())
	}

	readContent := new(bytes.Buffer)
	_, err = readContent.ReadFrom(reader)
	if err != nil {
		t.Error("error while reading test object: ", err.Error())
	}

	if bytes.Compare(content, readContent.Bytes()) != 0 {
		t.Error("retrieved test content differs")
	}

	err = ostore.DeleteObject(bucketName, objectName)
	if err != nil {
		t.Error("could not delete test object: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func TestObjectStore_SignedURL(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	objectName := "test/test.txt"
	content := []byte("this a great content!")

	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.PutObject(bucketName, objectName, bytes.NewReader(content))
	if err != nil {
		t.Error("could not create test object: ", err.Error())
	}

	url, err := ostore.GetSignedURL(bucketName, objectName, 10*time.Second)
	if err != nil {
		t.Error("could not get signed URL: ", err.Error())
	}

	if !strings.Contains(url, fmt.Sprintf("https://%s", bucketName)) {
		t.Error("signed URL is not correctly formatted")
	}

	err = ostore.DeleteObject(bucketName, objectName)
	if err != nil {
		t.Error("could not delete test object: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}
