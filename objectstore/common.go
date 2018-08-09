package objectstore

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
		"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/jinzhu/gorm"
)

// ManagedBucketNotFoundError signals that managed bucket was not found in database.
type ManagedBucketNotFoundError struct {
	errMessage string
}

func (err ManagedBucketNotFoundError) Error() string {
	return err.errMessage
}

func (ManagedBucketNotFoundError) NotFound() bool { return true }

// NewObjectStore creates a object store client for the given cloud type. The created object is initialized with
// the passed in secret and organization
func NewObjectStore(cloudType string, s *secret.SecretItemResponse, organization *auth.Organization) (objectstore.ObjectStore, error) {
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
		return azure.NewObjectStore(organization, s, database.GetDB(), log), nil
	case pkgCluster.Oracle:
		return &OCIObjectStore{
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

	if err := database.GetDB().Where(searchCriteria).Find(managedBucket).Error; err != nil {

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
	db := database.GetDB()
	return db.Save(m).Error
}

func updateDBField(m interface{}, field interface{}) error {
	log.Info("Updating Bucket Description ")
	db := database.GetDB()
	return db.Model(m).Update(field).Error
}

func deleteFromDbByPK(m interface{}) error {
	log.Info("Deleting from DB...")
	db := database.GetDB()
	return db.Delete(m).Error
}

func deleteFromDb(m interface{}) error {
	log.Info("Deleting from DB...")
	db := database.GetDB()
	return db.Delete(m, m).Error
}

// queryDb queries the database using the specified searchCriteria
// and returns the returned records into result
func queryWithOrderByDb(searchCriteria interface{}, orderBy interface{}, result interface{}) error {
	return database.GetDB().Where(searchCriteria).Order(orderBy).Find(result).Error
}
