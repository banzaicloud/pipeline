package objectstore

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin/json"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	api_storage "google.golang.org/api/storage/v1"
)

type GoogleObjectStore struct {
	location       string
	serviceAccount *cluster.ServiceAccount // TODO: serviceAccount type should be in a common place?
}

// TODO: this logic is duplicate thus should be in a common place so as it can be used from gke.go:newClientFromCredentials() as well
func NewGoogleServiceAccount(s *secret.SecretsItemResponse) *cluster.ServiceAccount {
	return &cluster.ServiceAccount{
		Type:                   s.Values[secret.Type],
		ProjectId:              s.Values[secret.ProjectId],
		PrivateKeyId:           s.Values[secret.PrivateKeyId],
		PrivateKey:             s.Values[secret.PrivateKey],
		ClientEmail:            s.Values[secret.ClientEmail],
		ClientId:               s.Values[secret.ClientId],
		AuthUri:                s.Values[secret.AuthUri],
		TokenUri:               s.Values[secret.TokenUri],
		AuthProviderX50CertUrl: s.Values[secret.AuthX509Url],
		ClientX509CertUrl:      s.Values[secret.ClientX509Url],
	}
}

func (b *GoogleObjectStore) CreateBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})
	ctx := context.Background()
	log.Info("Getting credentials")
	credentials, err := newGoogleCredentials(b)

	if err != nil {
		log.Errorf("Getting credentials failed due to: %s", err.Error())
		return err
	}

	log.Info("Creating new storage client")

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Errorf("Failed to create client: %s", err.Error())
		return err
	}
	defer client.Close()

	log.Info("Storage client created successfully")

	bucket := client.Bucket(bucketName)
	bucketAttrs := &storage.BucketAttrs{
		Location:      b.location,
		RequesterPays: false,
	}
	if err := bucket.Create(ctx, b.serviceAccount.ProjectId, bucketAttrs); err != nil {
		log.Errorf("Failed to create bucket: %s", err.Error())
		return err
	}
	log.Infof("%s bucket created in %s location", bucketName, b.location)
	return nil
}

func (b *GoogleObjectStore) DeleteBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "GoogleObjectStore.DeleteBucket"})
	ctx := context.Background()

	log.Info("Getting credentials")
	credentials, err := newGoogleCredentials(b)

	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return err
	}

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Errorf("Creating Google storage.Client failed: %s", err.Error())
		return err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName) // Which project should be billed for the operation, caller's or owners?

	if err := bucket.Delete(ctx); err != nil {
		return err
	}

	return nil
}

func (b *GoogleObjectStore) ListBuckets() error {
	return nil
}

func newGoogleCredentials(b *GoogleObjectStore) (*google.Credentials, error) {
	credentialsJson, err := json.Marshal(b.serviceAccount)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	credentials, err := google.CredentialsFromJSON(ctx, credentialsJson, api_storage.DevstorageFullControlScope)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}
