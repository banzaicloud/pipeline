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

package alibaba

import (
	"sort"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type AlibabaObjectStore struct {
	region string

	secret *secret.SecretItemResponse
	org    *auth.Organization
}

func NewObjectStore(region string, secret *secret.SecretItemResponse, org *auth.Organization) *AlibabaObjectStore {
	return &AlibabaObjectStore{
		region: region,
		secret: secret,
		org:    org,
	}
}

func (b *AlibabaObjectStore) CreateBucket(bucketName string) error {
	managedBucket := &ManagedAlibabaBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			return errors.Wrap(err, "error happened during getting bucket description from DB")
		}
	}

	log.Info("Creating AlibabaOSSClient...")
	svc, err := createAlibabaOSSClient(b.region, b.secret)
	if err != nil {
		return errors.Wrap(err, "Creating AlibabaOSSClient failed")
	}
	log.Info("AlibabaOSSClient create succeeded!")
	log.Debugf("Region is: %s", b.region)

	managedBucket.Name = bucketName
	managedBucket.Organization = *b.org
	managedBucket.Region = b.region

	if err = persistToDb(managedBucket); err != nil {
		return errors.Wrap(err, "Error happened during persisting bucket description to DB")
	}
	err = svc.CreateBucket(managedBucket.Name)
	if err != nil {
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}

		return errors.Wrap(err, "could not create a new OSS Bucket")
	}
	log.Debugf("Waiting for bucket %s to be created...", bucketName)

	// TODO: wait for bucket creation.
	log.Infof("Bucket %s Created", bucketName)

	return nil
}

func (b *AlibabaObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	svc, err := createAlibabaOSSClient(b.region, b.secret)
	if err != nil {
		return nil, err
	}

	log.Info("Retrieving bucket list from Alibaba")
	buckets, err := svc.ListBuckets()
	if err != nil {
		log.Errorf("Retrieving bucket list from Alibaba failed: %s", err.Error())
		return nil, err
	}

	log.Infof("Retrieving managed buckets")

	var managedAlibabaBuckets []ManagedAlibabaBucket
	if err = queryWithOrderByDb(&ManagedAlibabaBucket{OrgID: b.org.ID}, "name asc", &managedAlibabaBuckets); err != nil {
		log.Errorf("Retrieving managed buckets in organisation id=%s failed: %s", err.Error())
		return nil, err
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range buckets.Buckets {
		// managedAlibabaBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedAlibabaBuckets), func(i int) bool {
			return strings.Compare(managedAlibabaBuckets[i].Name, bucket.Name) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: bucket.Name, Managed: false}
		if idx < len(managedAlibabaBuckets) && strings.Compare(managedAlibabaBuckets[idx].Name, bucket.Name) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func (b *AlibabaObjectStore) DeleteBucket(bucketName string) error {
	managedBucket := &ManagedAlibabaBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)

	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return err
	}

	svc, err := createAlibabaOSSClient(managedBucket.Region, b.secret)
	if err != nil {
		log.Errorf("Creating OSSClient failed: %s", err.Error())
		return err
	}

	err = svc.DeleteBucket(managedBucket.Name)
	if err != nil {
		return err
	}

	// TODO: wait for bucket creation.
	if err = deleteFromDbByPK(managedBucket); err != nil {
		log.Errorf("Deleting managed OSS bucket from database failed: %s", err.Error())
		return err
	}

	return nil
}

func (b *AlibabaObjectStore) CheckBucket(bucketName string) error {
	svc, err := createAlibabaOSSClient(b.region, b.secret)

	if err != nil {
		log.Errorf("Creating AlibabaOSSClient failed: %s", err.Error())
		return errors.New("failed to create AlibabaOSSClient")
	}
	_, err = svc.GetBucketInfo(bucketName)
	if err != nil {
		log.Errorf("%s", err.Error())
		return err
	}

	return nil
}

// newManagedBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (b *AlibabaObjectStore) newManagedBucketSearchCriteria(bucketName string) *ManagedAlibabaBucket {
	return &ManagedAlibabaBucket{
		OrgID: b.org.ID,
		Name:  bucketName,
	}
}

func createAlibabaOSSClient(region string, retrievedSecret *secret.SecretItemResponse) (*oss.Client, error) {
	auth := verify.CreateAlibabaCredentials(retrievedSecret.Values)
	endpoint, err := ossRegionToEndpoint(region)
	if err != nil {
		return nil, err
	}
	return oss.New(endpoint, auth.AccessKeyId, auth.AccessKeySecret)
}

func ossRegionToEndpoint(region string) (endpoint string, err error) {
	switch region {
	case "oss-cn-hangzhou":
		endpoint = "https://oss-cn-hangzhou.aliyuncs.com"
	case "oss-cn-shanghai":
		endpoint = "https://oss-cn-shanghai.aliyuncs.com"
	case "oss-cn-qingdao":
		endpoint = "https://oss-cn-qingdao.aliyuncs.com"
	case "oss-cn-beijing":
		endpoint = "https://oss-cn-beijing.aliyuncs.com"
	case "oss-cn-zhangjiakou":
		endpoint = "https://oss-cn-zhangjiakou.aliyuncs.com"
	case "oss-cn-huhehaote":
		endpoint = "https://oss-cn-huhehaote.aliyuncs.com"
	case "oss-cn-shenzhen":
		endpoint = "https://oss-cn-shenzhen.aliyuncs.com"
	case "oss-cn-hongkong":
		endpoint = "https://oss-cn-hongkong.aliyuncs.com"
	case "oss-us-west-1":
		endpoint = "https://oss-us-west-1.aliyuncs.com"
	case "oss-us-east-1":
		endpoint = "https://oss-us-east-1.aliyuncs.com"
	case "oss-ap-southeast-1":
		endpoint = "https://oss-ap-southeast-1.aliyuncs.com"
	case "oss-ap-southeast-2":
		endpoint = "https://oss-ap-southeast-2.aliyuncs.com"
	case "oss-ap-southeast-3":
		endpoint = "https://oss-ap-southeast-3.aliyuncs.com"
	case "oss-ap-southeast-5":
		endpoint = "https://oss-ap-southeast-5.aliyuncs.com"
	case "oss-ap-northeast-1":
		endpoint = "https://oss-ap-northeast-1.aliyuncs.com"
	case "oss-ap-south-1":
		endpoint = "https://oss-ap-south-1.aliyuncs.com"
	case "oss-eu-central-1":
		endpoint = "https://oss-eu-central-1.aliyuncs.com"
	case "oss-me-east-1":
		endpoint = "https://oss-me-east-1.aliyuncs.com"
	default:
		err = errors.New("unknown endpoint")
	}

	return
}

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
