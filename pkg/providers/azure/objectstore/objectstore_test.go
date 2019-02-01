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
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/banzaicloud/pipeline/pkg/providers/azure"

	"github.com/pkg/errors"
)

const bucketName = "banzaicloud-test-bucket"
const nonExistingBucketName = "a-asd8908sad-nonexisting-bucketname"

func getObjectStore(t *testing.T) *objectStore {
	t.Helper()

	clientID := strings.TrimSpace(os.Getenv("AZURE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("AZURE_CLIENT_SECRET"))
	tenantID := strings.TrimSpace(os.Getenv("AZURE_TENANT_ID"))
	subscriptionID := strings.TrimSpace(os.Getenv("AZURE_SUBSCRIPTION_ID"))
	resourceGroup := strings.TrimSpace(os.Getenv("AZURE_RESOURCE_GROUP"))
	storageAccount := strings.TrimSpace(os.Getenv("AZURE_STORAGE_ACCOUNT"))

	if clientID == "" || clientSecret == "" || tenantID == "" || subscriptionID == "" || resourceGroup == "" || storageAccount == "" {
		t.Skip("missing necessary env variables")
	}

	config := Config{
		ResourceGroup:  resourceGroup,
		StorageAccount: storageAccount,
	}

	creds := azure.Credentials{
		ServicePrincipal: azure.ServicePrincipal{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TenantID:     tenantID,
		},
		SubscriptionID: subscriptionID,
	}

	ostore := New(config, creds)

	return ostore
}

func getBucketName(t *testing.T, bucketName string) string {
	t.Helper()

	prefix := strings.TrimSpace(os.Getenv("AZURE_BUCKET_PREFIX"))

	if prefix != "" {
		return fmt.Sprintf("%s-%s-%d", prefix, bucketName, time.Now().UnixNano())
	}

	return fmt.Sprintf("%s-%d", bucketName, time.Now().UnixNano())
}

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	t.Parallel()

	t.Run("ObjectStore_CreateAlreadyExistingBucket", testObjectStoreCreateAlreadyExistingBucket)
	t.Run("ObjectStore_BucketNotFound", testObjectStoreBucketNotFound)
	t.Run("ObjectStore_ObjectNotFound", testObjectStoreObjectNotFound)
	t.Run("ObjectStore_CreateDeleteBucket", testObjectStoreCreateDeleteBucket)
	t.Run("ObjectStore_ListBucket", testObjectStoreListBucket)
	t.Run("ObjectStore_CheckBucket", testObjectStoreCheckBucket)
	t.Run("ObjectStore_ListObjects", testObjectStoreListObjects)
	t.Run("ObjectStore_GetPutDeleteObject", testObjectStoreGetPutDeleteObject)
	t.Run("ObjectStore_SignedURL", testObjectStoreSignedURL)
}

func testObjectStoreCreateAlreadyExistingBucket(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.CreateBucket(bucketName)
	if _, ok := errors.Cause(err).(errBucketAlreadyExists); !ok {
		t.Error("error is not errBucketAlreadyExists: ", err.Error())
	}

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func testObjectStoreBucketNotFound(t *testing.T) {
	var err error

	ostore := getObjectStore(t)

	err = ostore.CheckBucket(nonExistingBucketName)
	if _, ok := errors.Cause(err).(errBucketNotFound); !ok {
		t.Fatal("error is not errBucketNotFound: ", err.Error())
	}
}

func testObjectStoreObjectNotFound(t *testing.T) {
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

func testObjectStoreCreateDeleteBucket(t *testing.T) {
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

func testObjectStoreListBucket(t *testing.T) {
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

func testObjectStoreCheckBucket(t *testing.T) {
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

func testObjectStoreListObjects(t *testing.T) {
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

func testObjectStoreGetPutDeleteObject(t *testing.T) {
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

func testObjectStoreSignedURL(t *testing.T) {
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

	if !strings.Contains(url, fmt.Sprintf("%s", bucketName)) {
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
