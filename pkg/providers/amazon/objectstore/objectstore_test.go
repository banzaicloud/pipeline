package objectstore

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func getSession(t *testing.T) *session.Session {
	t.Helper()

	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")
	region := os.Getenv("AWS_REGION")

	if accessKey == "" || secretKey == "" || region == "" {
		t.Skip("missing aws credentials")
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:      aws.String(region),
	})
	if err != nil {
		t.Fatal("could not create AWS session: ", err.Error())
	}

	return sess
}

func TestObjectStore_CreateBucket(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := New(sess, WaitForCompletion(true))

	bucketName := fmt.Sprintf("banzaicloud-test-bucket-%d", time.Now().UnixNano())

	err := s.CreateBucket(bucketName)
	if err != nil {
		t.Fatal("testing bucket creation failed: ", err.Error())
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

func TestObjectStore_ListBuckets(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := New(sess, WaitForCompletion(true))

	bucketName := fmt.Sprintf("banzaicloud-test-bucket-%d", time.Now().UnixNano())

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	buckets, err := s.ListBuckets()
	if err != nil {
		t.Error("testing bucket list failed: ", err.Error())
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

func TestObjectStore_CheckBucket(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := New(sess, WaitForCompletion(true))

	bucketName := fmt.Sprintf("banzaicloud-test-bucket-%d", time.Now().UnixNano())

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

func TestObjectStore_CheckBucket_DifferentRegion(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	// TODO: do not hardcode the region here
	s := New(sess.Copy(&aws.Config{Region: aws.String("eu-west-1")}), WaitForCompletion(true))

	bucketName := fmt.Sprintf("banzaicloud-test-bucket-%d", time.Now().UnixNano())

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

func TestObjectStore_Delete(t *testing.T) {
	sess := getSession(t)
	client := s3.New(sess)

	s := New(sess, WaitForCompletion(true))

	bucketName := fmt.Sprintf("banzaicloud-test-bucket-%d", time.Now().UnixNano())

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		t.Fatal("could not create test bucket: ", err.Error())
	}

	err = s.DeleteBucket(bucketName)
	if err != nil {
		t.Fatal("could not test bucket deletion: ", err.Error())
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
