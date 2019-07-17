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
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"emperror.dev/emperror"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const bucketName = "banzaicloud-test-bucket"
const nonExistingBucketName = "a-asd8908sad-nonexisting-bucketname"

func getObjectStore(t *testing.T) *objectStore {
	t.Helper()

	configFileLocation := os.Getenv("ORACLE_CLI_CONFIG_LOCATION")
	if configFileLocation == "" {
		t.Skip("Environment variable ORACLE_CLI_CONFIG_LOCATION is not set")
	}

	compartmentOCID := os.Getenv("ORACLE_COMPARTMENT_OCID")
	if compartmentOCID == "" {
		t.Skip("Compartment OCID is not set")
	}

	configuration, err := common.ConfigurationProviderFromFile(configFileLocation, "")
	if err != nil {
		t.Fatal(err)
	}

	region, _ := configuration.Region()
	UserOCID, _ := configuration.UserOCID()
	TenancyOCID, _ := configuration.TenancyOCID()
	privateRSAKey, _ := configuration.PrivateRSAKey()
	APIKeyFingerprint, _ := configuration.KeyFingerprint()

	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateRSAKey),
	}

	APIKey := pem.EncodeToMemory(privateKey)

	config := Config{
		Region: region,
	}

	credentials := Credentials{
		UserOCID:          UserOCID,
		TenancyOCID:       TenancyOCID,
		APIKey:            string(APIKey),
		APIKeyFingerprint: APIKeyFingerprint,
		CompartmentOCID:   compartmentOCID,
	}

	ostore, err := New(config, credentials)
	if err != nil {
		t.Fatal("could not create Oracle object storage client: ", err.Error())
	}

	return ostore
}

func getBucketName(t *testing.T, bucketName string) string {
	t.Helper()

	prefix := strings.TrimSpace(os.Getenv("ORACLE_BUCKET_PREFIX"))

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

	t.Run("ObjectStore_CreateDeleteBucket", testObjectStoreCreateDeleteBucket)
	t.Run("ObjectStore_ListBucket", testObjectStoreListBucket)
	t.Run("ObjectStore_CheckBucket", testObjectStoreCheckBucket)
	t.Run("ObjectStore_ListObjects", testObjectStoreListObjects)
	t.Run("ObjectStore_GetPutDeleteObject", testObjectStoreGetPutDeleteObject)
	t.Run("ObjectStore_SignedURL", testObjectStoreSignedURL)
	t.Run("ObjectStore_CreateAlreadyExistingBucket", testObjectStoreCreateAlreadyExistingBucket)
	t.Run("ObjectStore_BucketNotFound", testObjectStoreBucketNotFound)
	t.Run("ObjectStore_ObjectNotFound", testObjectStoreObjectNotFound)
	t.Run("ObjectStore_BucketErrorContext", testObjectStoreBucketErrorContext)
	t.Run("ObjectStore_ObjectErrorContext", testObjectStoreObjectErrorContext)
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
	assert.Exactly(t, objectNames, objects)

	objects, err = ostore.ListObjectsWithPrefix(bucketName, "test/")
	if err != nil {
		t.Error("could not list objects with prefix: ", err.Error())
	}

	sort.Strings(prefixedNames)
	sort.Strings(objects)
	assert.Exactly(t, prefixedNames, objects)

	_prefixes, err := ostore.ListObjectKeyPrefixes(bucketName, "/")
	if err != nil {
		t.Error("could not list object key prefixes: ", err.Error())
	}

	sort.Strings(prefixes)
	sort.Strings(_prefixes)
	assert.Exactly(t, prefixes, _prefixes)

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

	assert.Exactly(t, content, readContent.Bytes())

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

func testObjectStoreCreateAlreadyExistingBucket(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)
	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = ostore.CreateBucket(bucketName)
	assert.EqualError(t, errors.Cause(err), errBucketAlreadyExists{}.Error())

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func testObjectStoreBucketNotFound(t *testing.T) {
	var err error

	ostore := getObjectStore(t)

	err = ostore.CheckBucket(nonExistingBucketName)
	assert.EqualError(t, errors.Cause(err), errBucketNotFound{}.Error())
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
	assert.EqualError(t, errors.Cause(err), errObjectNotFound{}.Error())

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}

func testObjectStoreBucketErrorContext(t *testing.T) {
	var err error

	ostore := getObjectStore(t)

	err = ostore.CheckBucket(nonExistingBucketName)
	expected := []interface{}{"bucket", nonExistingBucketName}
	assert.Exactly(t, expected, emperror.Context(err))
}

func testObjectStoreObjectErrorContext(t *testing.T) {
	var err error

	bucketName := getBucketName(t, bucketName)

	ostore := getObjectStore(t)

	err = ostore.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	_, err = ostore.GetObject(bucketName, "test/test.txt")
	expected := []interface{}{"bucket", bucketName, "object", "test/test.txt"}
	assert.Exactly(t, expected, emperror.Context(err))

	err = ostore.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not delete test bucket: ", err.Error())
	}
}
