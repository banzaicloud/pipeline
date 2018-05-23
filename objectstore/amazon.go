package objectstore

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/sirupsen/logrus"
)

type AmazonObjectStore struct {
	bucketName string
	region     string
	secret     *secret.SecretsItemResponse
}

func (b *AmazonObjectStore) CreateBucket() error {
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
		Bucket: aws.String(b.bucketName),
	}
	_, err = svc.CreateBucket(input)
	if err != nil {
		log.Errorf("Could not create a new S3 Bucket, %s", err.Error())
		return err
	}
	log.Debugf("Waiting for bucket %s to be created...", b.bucketName)

	err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(b.bucketName),
	})
	if err != nil {
		log.Errorf("Error happened during waiting for the bucket to be created, %s", err.Error())
		return err
	}
	log.Infof("Bucket %s Created", b.bucketName)
	return nil
}

func (b *AmazonObjectStore) DeleteBucket() error {
	return nil
}

func (b *AmazonObjectStore) ListBuckets() error {
	return nil
}

func (b *AmazonObjectStore) createS3Client() (*s3.S3, error) {
	log := logger.WithFields(logrus.Fields{"tag": "createS3Client"})
	log.Info("Creating aws session")
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(b.region),
		Credentials:
			credentials.NewStaticCredentials(
				b.secret.Values[secret.AwsAccessKeyId],
				b.secret.Values[secret.AwsSecretAccessKey],
				""),
	})
	if err != nil {
		log.Errorf("Error creating AWS session %s", err.Error())
		return nil, err
	}
	log.Info("Aws session successfully created")
	return s3.New(s), nil
}
