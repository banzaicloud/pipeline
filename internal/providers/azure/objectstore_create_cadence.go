// +build cadence

package azure

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

// CreateBucket creates an Azure Object Store Blob with the provided name
// within a generated/provided ResourceGroup and StorageAccount
func (s *ObjectStore) CreateBucket(bucketName string) error {
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "error happened during getting bucket from DB: %s")
		}
	}

	// TODO: create the bucket in the database later so that we don't have to roll back
	bucket.ResourceGroup = resourceGroup
	bucket.Organization = *s.org

	logger.Info("saving bucket in DB")

	err := s.db.Save(bucket).Error
	if err != nil {
		return errors.Wrap(err, "error happened during saving bucket in DB")
	}

	// TODO: start cadence workflow here

	return nil
}
