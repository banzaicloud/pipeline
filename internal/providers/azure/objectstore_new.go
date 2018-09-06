// +build !cadence

package azure

import (
	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// ObjectStore stores all required parameters for container creation.
//
// Note: calling methods on this struct is not thread safe currently.
type ObjectStore struct {
	storageAccount string
	resourceGroup  string
	location       string
	secret         *secret.SecretItemResponse

	org *pipelineAuth.Organization

	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	location string,
	resourceGroup string,
	storageAccount string,
	secret *secret.SecretItemResponse,
	org *pipelineAuth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ObjectStore {
	return &ObjectStore{
		location:       location,
		resourceGroup:  resourceGroup,
		storageAccount: storageAccount,
		secret:         secret,
		db:             db,
		logger:         logger,
		org:            org,
	}
}
