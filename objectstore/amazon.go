package objectstore

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
)

type AmazonObjectStore struct {
	region string
	secret *secret.SecretsItemResponse
}

func (b *AmazonObjectStore) CreateBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})
	log.Info("Creating S3Client...")
	svc, err := b.createS3Client()
	if err != nil {
		log.Error("Creating S3Client failed!")
		return err
	}
	log.Info("S3Client create succeeded!")
	log.Debugf("Region is: %s", b.region)
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err = svc.CreateBucket(input)
	if err != nil {
		log.Errorf("Could not create a new S3 Bucket, %s", err.Error())
		return err
	}
	log.Debugf("Waiting for bucket %s to be created...", bucketName)

	err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		log.Errorf("Error happened during waiting for the bucket to be created, %s", err.Error())
		return err
	}
	log.Infof("Bucket %s Created", bucketName)
	return nil
}

func (b *AmazonObjectStore) DeleteBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "AmazonObjectStore.DeleteBucket"})

	svc, err := b.createS3Client()
	if err != nil {
		log.Error("Creating S3Client failed: %s", err.Error())
		return err
	}

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = svc.DeleteBucket(input)
	if err != nil {
		return err
	}

	err = svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		log.Errorf("Error occurred while waiting for the S3 Bucket to be deleted, %s", err.Error())
		return err
	}

	return nil
}

func (b *AmazonObjectStore) ListBuckets() error {
	return nil
}

func (b *AmazonObjectStore) createS3Client() (*s3.S3, error) {
	log := logger.WithFields(logrus.Fields{"tag": "createS3Client"})
	log.Info("Creating AWS session")
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(b.region),
		Credentials: credentials.NewStaticCredentials(
			b.secret.Values[secret.AwsAccessKeyId],
			b.secret.Values[secret.AwsSecretAccessKey],
			""),
	})

	if err != nil {
		log.Errorf("Error creating AWS session %s", err.Error())
		return nil, err
	}
	log.Info("AWS session successfully created")
	return s3.New(s), nil
}
