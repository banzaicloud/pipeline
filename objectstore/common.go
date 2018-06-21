package objectstore

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgStorage "github.com/banzaicloud/pipeline/pkg/storage"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

// ManagedBucketNotFoundError signals that managed bucket was not found in database.
type ManagedBucketNotFoundError struct {
	errMessage string
}

func (err ManagedBucketNotFoundError) Error() string {
	return err.errMessage
}

// ObjectStore is the interface that cloud specific object store implementation
// must implement
type ObjectStore interface {
	CreateBucket(string)
	ListBuckets() ([]*pkgStorage.BucketInfo, error)
	DeleteBucket(string) error
	CheckBucket(string) error

	WithResourceGroup(string) error
	WithStorageAccount(string) error
	WithRegion(string) error
}

// NewObjectStore creates a object store client for the given cloud type. The created object is initialized with
// the passed in secret and organization
func NewObjectStore(cloudType string, s *secret.SecretsItemResponse, organization *auth.Organization) (ObjectStore, error) {
	switch cloudType {
	case pkgCluster.Amazon:
		return &AmazonObjectStore{
			secret: s,
			org:    organization,
		}, nil
	case pkgCluster.Google:
		return &GoogleObjectStore{
			serviceAccount: verify.CreateServiceAccount(s.Values),
			org:            organization,
		}, nil
	case pkgCluster.Azure:
		return &AzureObjectStore{
			secret: s,
			org:    organization,
		}, nil
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}

// getManagedBucket looks up the managed bucket record in the database based on the specified
// searchCriteria and writes the db record into the managedBucket argument.
// If no db record is found than returns with ManagedBucketNotFoundError
func getManagedBucket(searchCriteria interface{}, managedBucket interface{}) error {

	if err := model.GetDB().Where(searchCriteria).Find(managedBucket).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return ManagedBucketNotFoundError{
				errMessage: err.Error(),
			}
		}
		return err
	}

	return nil
}

func persistToDb(m interface{}) error {
	log.Info("Persisting Bucket Description to Db")
	db := model.GetDB()
	return db.Save(m).Error
}

func updateDBField(m interface{}, field interface{}) error {
	log.Info("Updating Bucket Description ")
	db := model.GetDB()
	return db.Model(m).Update(field).Error
}

func deleteFromDbByPK(m interface{}) error {
	log.Info("Deleting from DB...")
	db := model.GetDB()
	return db.Delete(m).Error
}

func deleteFromDb(m interface{}) error {
	log.Info("Deleting from DB...")
	db := model.GetDB()
	return db.Delete(m, m).Error
}

// queryDb queries the database using the specified searchCriteria
// and returns the returned records into result
func queryWithOrderByDb(searchCriteria interface{}, orderBy interface{}, result interface{}) error {
	return model.GetDB().Where(searchCriteria).Order(orderBy).Find(result).Error
}
