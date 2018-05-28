package objectstore

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = config.Logger()
}

type CommonObjectStore interface {
	CreateBucket(string) error
	ListBuckets() error
	DeleteBucket(string) error
}

func ListCommonObjectStoreBuckets(s *secret.SecretsItemResponse) (CommonObjectStore, error) {
	switch s.SecretType {
	case constants.Amazon:
		return nil, nil
	case constants.Google:
		return nil, nil
	case constants.Azure:
		return nil, nil
	default:
		return nil, fmt.Errorf("listing a bucket is not supported for %s", s.SecretType)
	}
}

func CreateCommonObjectStoreBuckets(createBucketRequest components.CreateBucketRequest, s *secret.SecretsItemResponse) (CommonObjectStore, error) {
	switch s.SecretType {
	case constants.Amazon:
		return &AmazonObjectStore{
			region: createBucketRequest.Properties.CreateAmazonObjectStoreBucketProperties.Location,
			secret: s}, nil
	case constants.Google:
		return &GoogleObjectStore{
			location:       createBucketRequest.Properties.CreateGoogleObjectStoreBucketProperties.Location,
			serviceAccount: NewGoogleServiceAccount(s),
		}, nil
	case constants.Azure:
		return &AzureObjectStore{
			storageAccount: createBucketRequest.Properties.CreateAzureObjectStoreBucketProperties.StorageAccount,
			resourceGroup:  createBucketRequest.Properties.CreateAzureObjectStoreBucketProperties.ResourceGroup,
			location:       createBucketRequest.Properties.CreateAzureObjectStoreBucketProperties.Location,
			secret:         s}, nil
	default:
		return nil, fmt.Errorf("creating a bucket is not supported for %s", s)
	}
}

func NewGoogleObjectStore(s *secret.SecretsItemResponse) (CommonObjectStore, error) {
	return &GoogleObjectStore{
		serviceAccount: NewGoogleServiceAccount(s),
	}, nil
}

func NewAmazonObjectStore(s *secret.SecretsItemResponse, region string) (CommonObjectStore, error) {
	return &AmazonObjectStore{
		secret: s,
		region: region,
	}, nil
}

func NewAzureObjectStore(s *secret.SecretsItemResponse, resourceGroup, storageAccount string) (CommonObjectStore, error) {
	return &AzureObjectStore{
		storageAccount: storageAccount,
		resourceGroup:  resourceGroup,
		secret:         s,
	}, nil
}
