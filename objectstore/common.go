package objectstore

import (
	"github.com/banzaicloud/banzai-types/constants"
	"fmt"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/banzai-types/components"
)
var logger *logrus.Logger
func init() {
	logger = config.Logger()
}

type CommonObjectStore interface {
	CreateBucket() error
	DeleteBucket() error
	ListBuckets() error
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
	switch s.SecretType{
	case constants.Amazon:
		return &AmazonObjectStore{
			bucketName: createBucketRequest.Properties.Name,
			region: createBucketRequest.Properties.Location,
			secret: s}, nil
	case constants.Google:
		return &GoogleObjectStore{
			bucketName: createBucketRequest.Properties.Name,
			projectId: s.Values["project_id"]}, nil
	case constants.Azure:
		return &AzureObjectStore{
			bucketName: createBucketRequest.Properties.Name,
			storageAccount: createBucketRequest.Properties.StorageAccount,
			resourceGroup: createBucketRequest.Properties.ResourceGroup,
			location: createBucketRequest.Properties.Location,
			secret: s,}, nil
	default:
		return nil, fmt.Errorf("creating a bucket is not supported for %s", s)
	}
}
