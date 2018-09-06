// +build cadence

package azure

import (
	"context"
	"time"

	"github.com/banzaicloud/pipeline/config"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
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

	bucket.Organization = *s.org
	bucket.ResourceGroup = resourceGroup
	bucket.StorageAccount = storageAccount
	bucket.Location = s.location
	bucket.Name = bucketName

	logger.Info("saving bucket in DB")

	err := s.db.Save(bucket).Error
	if err != nil {
		return errors.Wrap(err, "error happened during saving bucket in DB")
	}

	workflowContext := CreateBucketWorkflowContext{
		OrganizationID: s.org.ID,
		SecretID:       s.secret.ID,
		Location:       s.location,
		ResourceGroup:  resourceGroup,
		StorageAccount: storageAccount,
		Bucket:         bucketName,
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     config.CadenceTaskList(),
		ExecutionStartToCloseTimeout: 10 * time.Minute, // TODO: lower timeout
	}

	exec, err := s.workflowClient.StartWorkflow(context.Background(), workflowOptions, CreateBucketWorkflowType, workflowContext)
	if err != nil {
		return errors.Wrap(err, "could not start workflow")
	}

	logger.WithFields(logrus.Fields{
		"workflow-id": exec.ID,
		"run-id":      exec.RunID,
	}).Info("started workflow")

	return nil
}
