// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "error happened during getting bucket from DB: %s")
		}
	}

	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

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
		BucketID:       bucket.ID,
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
