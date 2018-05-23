package objectstore

import (
	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"context"
)

type GoogleObjectStore struct {
	bucketName string
	projectId string
}

func (b *GoogleObjectStore) CreateBucket() error {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})
	ctx := context.Background()
	log.Info("Creating new storage client")
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf("Failed to create client: %s", err.Error())
		return err
	}
	log.Info("Storage client created successfully")

	bucket := client.Bucket(b.bucketName)
	if err := bucket.Create(ctx, b.projectId, nil); err != nil {
		log.Errorf("Failed to create bucket: %s", err.Error())
		return err
	}
	log.Infof("%s bucket created", b.bucketName)
	return nil
}

func (b *GoogleObjectStore) DeleteBucket() error {
	return nil
}

func (b *GoogleObjectStore) ListBuckets() error {
	return nil
}
