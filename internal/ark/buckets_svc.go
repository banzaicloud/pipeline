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

package ark

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/heptio/ark/pkg/cloudprovider"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/pkg/apis/core"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/src/auth"
)

// BucketsService is for buckets related ARK functions
type BucketsService struct {
	org        *auth.Organization
	repository *BucketsRepository
	logger     logrus.FieldLogger
}

// BucketsServiceFactory creates and returns an initialized BucketsService instance
func BucketsServiceFactory(org *auth.Organization, db *gorm.DB, logger logrus.FieldLogger) *BucketsService {

	return NewBucketsService(org, NewBucketsRepository(org, db, logger), logger)
}

// NewBucketsService creates and returns an initialized BucketsService instance
func NewBucketsService(
	org *auth.Organization,
	repository *BucketsRepository,
	logger logrus.FieldLogger,
) *BucketsService {

	return &BucketsService{
		org:        org,
		repository: repository,
		logger:     logger,
	}
}

// GetObjectStoreForBucket create an initialized ObjectStore
func (s *BucketsService) GetObjectStoreForBucket(bucket *api.Bucket) (cloudprovider.ObjectStore, error) {
	if bucket == nil {
		return nil, errors.New("could not get object store, bucket is nil")
	}

	secret, err := GetSecretWithValidation(bucket.SecretID, s.org.ID, bucket.Cloud)
	if err != nil {
		return nil, errors.Wrap(err, "could not get secret with validation")
	}

	ctx := providers.ObjectStoreContext{
		Provider:       bucket.Cloud,
		Secret:         secret,
		Location:       bucket.Location,
		StorageAccount: bucket.StorageAccount,
		ResourceGroup:  bucket.ResourceGroup,
	}

	os, err := NewObjectStore(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize object store client")
	}

	return os, nil
}

// GetBackupsFromObjectStore gets Backups from object store bucket
func (s *BucketsService) GetBackupsFromObjectStore(bucket *api.Bucket) ([]*arkAPI.Backup, error) {

	os, err := s.GetObjectStoreForBucket(bucket)
	if err != nil {
		return nil, err
	}

	svc := cloudprovider.NewBackupService(os, s.logger)
	backups, err := svc.GetAllBackups(bucket.Name)
	if err != nil {
		return nil, err
	}

	return backups, nil
}

// GetActiveDeploymentModel gets the active ARK ClusterBackupDeploymentsModel
func (s *BucketsService) GetActiveDeploymentModel(bucket *ClusterBackupBucketsModel) (
	ClusterBackupDeploymentsModel, error) {

	return s.repository.GetActiveDeploymentModel(bucket)
}

// GetModels gets ClusterBackupBucketsModels
func (s *BucketsService) GetModels() ([]*ClusterBackupBucketsModel, error) {

	return s.repository.Find()
}

// GetByRequest finds a Bucket by a FindBucketRequest
func (s *BucketsService) GetByRequest(req api.FindBucketRequest) (*api.Bucket, error) {

	bucket, err := s.repository.FindOneByRequest(req)
	if err != nil {
		return nil, err
	}

	return bucket.ConvertModelToEntity(), nil
}

// GetByID gets a Bucket by an id
func (s *BucketsService) GetByID(id uint) (*api.Bucket, error) {

	bucket, err := s.repository.FindOneByID(id)
	if err != nil {
		return nil, err
	}

	return bucket.ConvertModelToEntity(), nil
}

// GetByName gets a Bucket by name
func (s *BucketsService) GetByName(name string) (*api.Bucket, error) {

	bucket, err := s.repository.FindOneByName(name)
	if err != nil {
		return nil, err
	}

	return bucket.ConvertModelToEntity(), nil
}

// DeleteByName deletes a Bucket by name
func (s *BucketsService) DeleteByName(name string) error {

	bucket, err := s.repository.FindOneByName(name)
	if err != nil {
		return err
	}

	if bucket.Deployment.Cluster.ID > 0 {
		return errors.New("bucket is in use")
	}

	return s.repository.Delete(bucket)
}

// DeleteByName deletes a Bucket by ID
func (s *BucketsService) DeleteByID(id uint) error {

	bucket, err := s.repository.FindOneByID(id)
	if err != nil {
		return err
	}

	if bucket.Deployment.Cluster.ID > 0 {
		return errors.New("bucket is in use")
	}

	return s.repository.Delete(bucket)
}

// List gets all Buckets
func (s *BucketsService) List() ([]*api.Bucket, error) {

	buckets := make([]*api.Bucket, 0)

	items, err := s.GetModels()
	if err != nil {
		return buckets, err
	}

	for _, item := range items {
		bucket := item.ConvertModelToEntity()
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// IsBucketInUse check whether a ClusterBackupBucketsModel is used in an active ARK deployment
func (s *BucketsService) IsBucketInUse(bucket *ClusterBackupBucketsModel) error {

	return s.repository.IsInUse(bucket)
}

// FindOrCreateBucket finds or create a new ClusterBackupBucketsModel by a CreateBucketRequest
func (s *BucketsService) FindOrCreateBucket(req *api.CreateBucketRequest) (*ClusterBackupBucketsModel, error) {

	err := ValidateCreateBucketRequest(req, s.org)
	if err != nil {
		return nil, err
	}

	bucket, err := s.repository.FindOneOrCreateByRequest(req)
	if err != nil {
		return nil, err
	}

	return bucket, err
}

// GetNodesFromBackupContents gets core.NodeList from a backup in an object store bucket
func (s *BucketsService) GetNodesFromBackupContents(bucket *api.Bucket, backupName string) (
	nodes core.NodeList, err error) {

	nodes.APIVersion = "v1"

	buf := new(bytes.Buffer)
	err = s.StreamBackupContentsFromObjectStore(bucket, backupName, buf)

	if err != nil {
		s.logger.Error(err.Error())
		return nodes, err
	}

	gzf, err := gzip.NewReader(buf)
	if err != nil {
		return
	}

	tarReader := tar.NewReader(gzf)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.logger.Error(err)
			return nodes, err
		}

		r, _ := regexp.Compile(`resources/nodes/cluster/[a-z0-9-.]+\.json`)
		name := header.Name

		var node core.Node
		if header.Typeflag == tar.TypeReg && r.MatchString(header.Name) {
			s.logger.WithField("filename", name).Debug("node backup found")
			nodeBuf := new(bytes.Buffer)
			_, err := io.Copy(nodeBuf, tarReader)
			if err != nil {
				return nodes, err
			}
			err = json.Unmarshal(nodeBuf.Bytes(), &node)
			if err != nil {
				return nodes, err
			}
			nodes.Items = append(nodes.Items, node)
		}
	}

	return nodes, err
}

// StreamRestoreResultsFromObjectStore streams a restore result from object store to the given io.Writer
func (s *BucketsService) StreamRestoreResultsFromObjectStore(
	bucket *api.Bucket,
	backupName string,
	restoreName string,
	w io.Writer,
) error {

	return s.streamObjectFromObjectStore(arkAPI.DownloadTarget{
		Kind: arkAPI.DownloadTargetKindRestoreResults,
		Name: restoreName,
	}, bucket, backupName, w)
}

// StreamRestoreLogsFromObjectStore streams a restore logs from object store to the given io.Writer
func (s *BucketsService) StreamRestoreLogsFromObjectStore(
	bucket *api.Bucket,
	backupName string,
	restoreName string,
	w io.Writer,
) error {

	return s.streamObjectFromObjectStore(arkAPI.DownloadTarget{
		Kind: arkAPI.DownloadTargetKindRestoreLog,
		Name: restoreName,
	}, bucket, backupName, w)
}

// StreamBackupLogsFromObjectStore streams a backup logs from object store to the given io.Writer
func (s *BucketsService) StreamBackupLogsFromObjectStore(
	bucket *api.Bucket,
	backupName string,
	w io.Writer,
) error {

	return s.streamObjectFromObjectStore(arkAPI.DownloadTarget{
		Kind: arkAPI.DownloadTargetKindBackupLog,
		Name: backupName,
	}, bucket, backupName, w)
}

// StreamBackupContentsFromObjectStore streams a backup contents from object store to the given io.Writer
func (s *BucketsService) StreamBackupContentsFromObjectStore(
	bucket *api.Bucket,
	backupName string,
	w io.Writer,
) error {

	return s.streamObjectFromObjectStore(arkAPI.DownloadTarget{
		Kind: arkAPI.DownloadTargetKindBackupContents,
		Name: backupName,
	}, bucket, backupName, w)
}

func (s *BucketsService) streamObjectFromObjectStore(
	target arkAPI.DownloadTarget,
	bucket *api.Bucket,
	backupName string,
	w io.Writer,
) error {

	os, err := s.GetObjectStoreForBucket(bucket)
	if err != nil {
		return err
	}

	svc := cloudprovider.NewBackupService(os, s.logger)

	url, err := svc.CreateSignedURL(target, bucket.Name, backupName, 10*time.Minute)
	if err != nil {
		return err
	}

	httpClient := new(http.Client)
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Manually set this header so the net/http library does not automatically try to decompress. We
	// need to handle this manually because it's not currently possible to set the MIME type for the
	// pre-signed URLs for GCP or Azure.
	httpReq.Header.Set("Accept-Encoding", "gzip")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "request failed: unable to decode response body")
		}

		return errors.Errorf("request failed: %v", string(body))
	}

	reader := resp.Body
	if target.Kind != arkAPI.DownloadTargetKindBackupContents {
		// need to decompress logs
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	_, err = io.Copy(w, reader)
	return err
}
