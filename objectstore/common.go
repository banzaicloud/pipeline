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

package objectstore

import (
	"github.com/banzaicloud/pipeline/config"
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

// getManagedBucket looks up the managed bucket record in the database based on the specified
// searchCriteria and writes the db record into the managedBucket argument.
// If no db record is found than returns with ManagedBucketNotFoundError
func getManagedBucket(searchCriteria interface{}, managedBucket interface{}) error {

	if err := config.DB().Where(searchCriteria).Find(managedBucket).Error; err != nil {

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
	log.Info("Persisting Bucket to Db")
	db := config.DB()
	return db.Save(m).Error
}

func deleteFromDbByPK(m interface{}) error {
	log.Info("Deleting from DB...")
	db := config.DB()
	return db.Delete(m).Error
}

// queryDb queries the database using the specified searchCriteria
// and returns the returned records into result
func queryWithOrderByDb(searchCriteria interface{}, orderBy interface{}, result interface{}) error {
	return config.DB().Where(searchCriteria).Order(orderBy).Find(result).Error
}
