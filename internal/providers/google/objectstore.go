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

package google

import (
	"sort"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	googleObjectstore "github.com/banzaicloud/pipeline/pkg/providers/google/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

type googleObjectStore interface {
	commonObjectstore.ObjectStore
}

// ObjectStore stores all required parameters for bucket creation.
type ObjectStore struct {
	objectStore googleObjectStore

	db     *gorm.DB
	logger logrus.FieldLogger

	org            *auth.Organization
	serviceAccount *verify.ServiceAccount
	secret         *secret.SecretItemResponse

	location string
	force    bool
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	org *auth.Organization,
	secret *secret.SecretItemResponse,
	location string,
	db *gorm.DB,
	logger logrus.FieldLogger,
	force bool,
) (*ObjectStore, error) {
	var serviceAccount *verify.ServiceAccount
	if secret != nil {
		serviceAccount = verify.CreateServiceAccount(secret.Values)
	}
	ostore, err := getProviderObjectStore(secret, location)
	if err != nil {
		return nil, errors.Wrap(err, "could not create Google object storage client")
	}

	return &ObjectStore{
		objectStore:    ostore,
		db:             db,
		logger:         logger,
		org:            org,
		secret:         secret,
		serviceAccount: serviceAccount,
		location:       location,
		force:          force,
	}, nil
}

func getProviderObjectStore(secret *secret.SecretItemResponse, location string) (googleObjectStore, error) {
	// when no secrets provided build an object store with no provider client/session setup
	// eg. usage: list managed buckets
	if secret == nil {
		return googleObjectstore.NewPlainObjectStore()
	}

	config := googleObjectstore.Config{
		Region: location,
	}

	credentials := googleObjectstore.Credentials{
		Type:                   secret.Values[pkgSecret.Type],
		ProjectID:              secret.Values[pkgSecret.ProjectId],
		PrivateKeyID:           secret.Values[pkgSecret.PrivateKeyId],
		PrivateKey:             secret.Values[pkgSecret.PrivateKey],
		ClientEmail:            secret.Values[pkgSecret.ClientEmail],
		ClientID:               secret.Values[pkgSecret.ClientId],
		AuthURI:                secret.Values[pkgSecret.AuthUri],
		TokenURI:               secret.Values[pkgSecret.TokenUri],
		AuthProviderX50CertURL: secret.Values[pkgSecret.AuthX509Url],
		ClientX509CertURL:      secret.Values[pkgSecret.ClientX509Url],
	}

	ostore, err := googleObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return ostore, nil
}

func (s *ObjectStore) getLogger() logrus.FieldLogger {
	var sId string
	if s.secret == nil {
		sId = ""
	} else {
		sId = s.secret.ID
	}

	return s.logger.WithFields(logrus.Fields{
		"organization": s.org.ID,
		"secret":       sId,
		"location":     s.location,
	})
}

// CreateBucket creates a Google Bucket with the provided name and location.
func (s *ObjectStore) CreateBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	// lookup the bucket in the db
	dbr := s.db.Where(searchCriteria).Find(bucket)

	switch dbr.Error {
	case nil:
		return emperror.WrapWith(dbr.Error, "the bucket already exists", "bucket", bucketName)
	case gorm.ErrRecordNotFound:
		// proceed to creation
	default:
		return emperror.WrapWith(dbr.Error, "failed to retrieve bucket", "bucket", bucketName)
	}

	bucket.Name = bucketName
	bucket.Organization = *s.org
	bucket.Location = s.location
	bucket.SecretRef = s.secret.ID
	bucket.Status = providers.BucketCreating

	logger.Info("creating bucket...")

	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucketName)
	}

	if err := s.objectStore.CreateBucket(bucketName); err != nil {
		bucket.Status = providers.BucketCreateError
		bucket.StatusMsg = err.Error()
		if e := s.db.Save(bucket).Error; e != nil {
			return emperror.WrapWith(e, "failed to persist the bucket", "bucket", bucketName)
		}
		return emperror.WrapWith(err, "failed to create the bucket", "bucket", bucketName)
	}

	bucket.Status = providers.BucketCreated
	bucket.StatusMsg = "bucket successfully created"
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to persist the bucket", "bucket", bucketName)
	}
	logger.Info("bucket created")

	return nil
}

// DeleteBucket deletes the GS bucket identified by the specified name
// provided the storage container is of 'managed' type.
func (s *ObjectStore) DeleteBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking up the bucket...")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	if err := s.deleteFromProvider(bucket); err != nil {
		if !s.force {
			// if delete is not forced return here
			return s.deleteFailed(bucket, err)
		}
	}

	if err := s.db.Delete(bucket).Error; err != nil {
		return s.deleteFailed(bucket, err)
	}

	return nil
}

func (s *ObjectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {
	logger := s.getLogger().WithField("bucket", bucket.Name)
	logger.Info("deleting bucket on provider...")

	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to update bucket", "bucket", bucket.Name)
	}

	objectStore, err := getProviderObjectStore(s.secret, bucket.Location)
	if err != nil {
		return emperror.WrapWith(err, "failed to create object store", "bucket", bucket.Name)
	}

	if err := objectStore.DeleteBucket(bucket.Name); err != nil {
		return emperror.WrapWith(err, "failed to delete bucket from provider", "bucket", bucket.Name)
	}

	return nil
}

func (s *ObjectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {
	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to delete bucket", "bucket", bucket.Name)
	}
	return reason
}

// CheckBucket checks the status of the given Google bucket.
func (s *ObjectStore) CheckBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)
	logger.Info("looking up the bucket...")

	if err := s.objectStore.CheckBucket(bucketName); err != nil {
		return emperror.WrapWith(err, "failed to check the bucket", "bucket", bucketName)
	}

	return nil
}

// ListBuckets returns a list of GS buckets that can be accessed with the credentials
// referenced by the secret field. GS buckets that were created by a user in the current
// org are marked as 'managed`
func (s *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()

	logger.Info("retrieving buckets from provider...")
	GSBuckets, err := s.objectStore.ListBuckets()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve buckets")
	}

	logger.Info("retrieving managed buckets...")
	var managedBuckets []ObjectStoreBucketModel

	err = s.db.Where(ObjectStoreBucketModel{OrganizationID: s.org.ID}).Order("name asc").Find(&managedBuckets).Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range GSBuckets {
		// managedBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedBuckets), func(i int) bool {
			return strings.Compare(managedBuckets[i].Name, bucket) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: bucket, Managed: false}
		if idx < len(managedBuckets) && strings.Compare(managedBuckets[idx].Name, bucket) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func (s *ObjectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()
	logger.Debug("retrieving managed bucket list")

	var googleBuckets []ObjectStoreBucketModel

	if err := s.db.Where(ObjectStoreBucketModel{OrganizationID: s.org.ID}).Order("name asc").Find(&googleBuckets).Error; err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range googleBuckets {
		bucketList = append(bucketList, &objectstore.BucketInfo{
			Name:      bucket.Name,
			Managed:   true,
			Location:  bucket.Location,
			SecretRef: bucket.SecretRef,
			Cloud:     providers.Google,
			Status:    bucket.Status,
			StatusMsg: bucket.StatusMsg,
		})
	}

	return bucketList, nil
}

// searchCriteria returns the database search criteria to find managed bucket with the given name.
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
	}
}
