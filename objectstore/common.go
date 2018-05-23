package objectstore

import (
	"github.com/banzaicloud/banzai-types/constants"
	"fmt"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/config"
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

func CreateCommonObjectStoreBuckets(name string, s *secret.SecretsItemResponse) (CommonObjectStore, error) {
	switch s.SecretType{
	case constants.Amazon:
		return &AmazonObjectStore{}, nil
	case constants.Google:
		return &GoogleObjectStore{bucketName: name, projectId: s.Values["project_id"]}, nil
	case constants.Azure:
		return &AzureObjectStore{}, nil
	default:
		return nil, fmt.Errorf("creating a bucket is not supported for %s", s)
	}
}
