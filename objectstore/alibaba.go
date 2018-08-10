package objectstore

import (
	"errors"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
)

// ManagedAlibabaBucket is the schema for the DB
type ManagedAlibabaBucket struct {
	ID           uint              `gorm:"primary_key"`
	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`
	Name         string            `gorm:"unique_index:bucketName"`
	Region       string
}

type AlibabaObjectStore struct {
	region string
	secret *secret.SecretItemResponse
	org    *auth.Organization
}

var _ objectstore.ObjectStore = (*AlibabaObjectStore)(nil)

func (b *AlibabaObjectStore) CreateBucket(bucketName string) {
	managedBucket := &ManagedAlibabaBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			log.Errorf("Error happened during getting bucket description from DB %s", err.Error())
			return
		}
	}

	log.Info("Creating AlibabaOSSClient...")
	svc, err := createAlibabaOSSClient(b.region, b.secret)
	if err != nil {
		log.Error("Creating AlibabaOSSClient failed!")
		return
	}
	log.Info("AlibabaOSSClient create succeeded!")
	log.Debugf("Region is: %s", b.region)

	managedBucket.Name = bucketName
	managedBucket.Organization = *b.org
	managedBucket.Region = b.region

	if err = persistToDb(managedBucket); err != nil {
		log.Errorf("Error happened during persisting bucket description to DB %s", err.Error())
		return
	}
	err = svc.CreateBucket(managedBucket.Name)
	if err != nil {
		log.Errorf("Could not create a new OSS Bucket, %s", err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	log.Debugf("Waiting for bucket %s to be created...", bucketName)

	// TODO: wait for bucket creation.
	log.Infof("Bucket %s Created", bucketName)
	return
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
	managedBucket := &ManagedAlibabaBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return ManagedBucketNotFoundError{}
	}

	svc, err := createAlibabaOSSClient(managedBucket.Region, b.secret)

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

func (AlibabaObjectStore) WithResourceGroup(string) error {
	return nil
}

func (AlibabaObjectStore) WithStorageAccount(string) error {
	return nil
}

func (b *AlibabaObjectStore) WithRegion(region string) error {
	b.region = region
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
