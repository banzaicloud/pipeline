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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

const (
	bucketName            = "banzaicloud-test-bucket"
	nonExistingBucketName = "a-asd8908sad-nonexisting-bucketname"
)

func getObjectStore(t *testing.T) *objectStore {
	t.Helper()

	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		t.Skip("missing region")
	}

	return getObjectStoreWithRegion(t, region)
}

func getObjectStoreWithRegion(t *testing.T, region string) *objectStore {
	t.Helper()

	accessKey := strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("AWS_SECRET_KEY"))

	if accessKey == "" || secretKey == "" {
		t.Skip("missing credentials")
	}

	credentials := Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}

	config := Config{
		Region: region,
		Opts: []Option{
			WaitForCompletion(true),
		},
	}

	os, err := New(config, credentials)
	if err != nil {
		t.Fatal("could not create object storage client: ", err.Error())
	}

	return os
}

func getSession(t *testing.T) *session.Session {
	t.Helper()

	accessKey := strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("AWS_SECRET_KEY"))
	region := strings.TrimSpace(os.Getenv("AWS_REGION"))

	if accessKey == "" || secretKey == "" || region == "" {
		t.Skip("missing credentials")
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:      aws.String(region),
	})
	if err != nil {
		t.Fatal("could not create session: ", err.Error())
	}

	return sess
}

func getBucketName(t *testing.T, bucketName string) string {
	t.Helper()

	prefix := strings.TrimSpace(os.Getenv("AWS_BUCKET_PREFIX"))

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

	t.Run("ObjectStore_CreateBucket", testObjectStoreCreateBucket)
	t.Run("ObjectStore_GetRegion", testObjectStoreGetRegion)
	t.Run("ObjectStore_ListBuckets", testObjectStoreListBuckets)
	t.Run("ObjectStore_CheckBucket", testObjectStoreCheckBucket)
	t.Run("ObjectStore_CheckBucket_DifferentRegion", testObjectStoreCheckBucketDifferentRegion)
	t.Run("ObjectStore_Delete", testObjectStoreDelete)
	t.Run("ObjectStore_ListObjects", testObjectStoreListObjects)
	t.Run("ObjectStore_GetPutDeleteObject", testObjectStoreGetPutDeleteObject)
	t.Run("ObjectStore_SignedURL", testObjectStoreSignedURL)
	t.Run("ObjectStore_CreateAlreadyExistingBucket", testObjectStoreCreateAlreadyExistingBucket)
	t.Run("ObjectStore_BucketNotFound", testObjectStoreBucketNotFound)
	t.Run("ObjectStore_ObjectNotFound", testObjectStoreObjectNotFound)
}

func testObjectStoreCreateBucket(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := getObjectStore(t)

	bucketName := getBucketName(t, bucketName)

	err := s.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	head := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.HeadBucket(head)
	if err != nil {
		t.Error("could not verify bucket creation: ", err.Error())
	}

	del := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.DeleteBucket(del)
	if err != nil {
		t.Fatal("could not clean up bucket: ", err.Error())
	}
}

func testObjectStoreGetRegion(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := getObjectStore(t)

	bucketName := getBucketName(t, bucketName)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	region, err := s.GetRegion(bucketName)
	if err != nil {
		t.Error("could not get bucket region: ", err.Error())
	} else {
		if strings.TrimSpace(os.Getenv("AWS_REGION")) != region {
			t.Error("test bucket region does not match")
		}
	}

	del := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.DeleteBucket(del)
	if err != nil {
		t.Fatal("could not clean up bucket: ", err.Error())
	}
}

func testObjectStoreListBuckets(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := getObjectStore(t)

	bucketName := getBucketName(t, bucketName)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	buckets, err := s.ListBuckets()
	if err != nil {
		t.Error("could not list buckets: ", err.Error())
	} else {
		var bucketFound bool

		for _, bucket := range buckets {
			if bucket == bucketName {
				bucketFound = true

				break
			}
		}

		if !bucketFound {
			t.Error("test bucket not found in the list")
		}
	}

	del := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.DeleteBucket(del)
	if err != nil {
		t.Fatal("could not clean up bucket: ", err.Error())
	}
}

func testObjectStoreCheckBucket(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := getObjectStore(t)

	bucketName := getBucketName(t, bucketName)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = s.CheckBucket(bucketName)
	if err != nil {
		t.Error("bucket checking failed: ", err.Error())
	}

	del := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.DeleteBucket(del)
	if err != nil {
		t.Fatal("could not clean up bucket: ", err.Error())
	}
}

func testObjectStoreCheckBucketDifferentRegion(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	diffRegion := strings.TrimSpace(os.Getenv("AWS_DIFF_REGION"))
	if diffRegion == "" {
		t.Skip("no different region was set")
	}
	if diffRegion == *sess.Config.Region {
		t.Skip("same regions were set")
	}

	s := getObjectStoreWithRegion(t, diffRegion)

	bucketName := getBucketName(t, bucketName)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = s.CheckBucket(bucketName)
	if err != nil {
		t.Error("checking bucket failed: ", err.Error())
	}

	del := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.DeleteBucket(del)
	if err != nil {
		t.Fatal("could not clean up bucket: ", err.Error())
	}
}

func testObjectStoreDelete(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := getObjectStore(t)

	bucketName := getBucketName(t, bucketName)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = s.DeleteBucket(bucketName)
	if err != nil {
		// this test seems to be the most flaky one, give it another chance
		time.Sleep(time.Second)
		err = s.DeleteBucket(bucketName)
		if err != nil {
			t.Fatal("could not test bucket deletion: ", err.Error())
		}
	}

	head := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.HeadBucket(head)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); !ok || awsErr.Code() != "NotFound" {
			t.Error("could not verify bucket deletion: ", err.Error())
		}
	} else {
		t.Error("could not verify bucket deletion: no error received")
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
