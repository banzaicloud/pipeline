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
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	apiStorage "google.golang.org/api/storage/v1"
)

type objectStore struct {
	config      Config
	credentials Credentials

	client *storage.Client

	projectID      string
	googleAccessID string
	privateKey     []byte
}

// Config defines configuration
type Config struct {
	Region string
}

// Credentials represents credentials necessary for access
type Credentials struct {
	Type                   string `json:"type"`
	ProjectID              string `json:"project_id"`
	PrivateKeyID           string `json:"private_key_id"`
	PrivateKey             string `json:"private_key"`
	ClientEmail            string `json:"client_email"`
	ClientID               string `json:"client_id"`
	AuthURI                string `json:"auth_uri"`
	TokenURI               string `json:"token_uri"`
	AuthProviderX50CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL      string `json:"client_x509_cert_url"`
}

// New returns an Object Store instance that manages Google object storage
func New(config Config, credentials Credentials) (*objectStore, error) {
	o := &objectStore{
		credentials: credentials,
		config:      config,

		projectID: credentials.ProjectID,
	}

	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return nil, emperror.Wrap(err, "could not marshal credentials")
	}

	jwtConfig, err := google.JWTConfigFromJSON(credentialsJSON)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get JWT config from JSON")
	}
	if jwtConfig.Email == "" {
		return nil, emperror.Wrap(err, "credentials does not contain an email")
	}
	if len(jwtConfig.PrivateKey) == 0 {
		return nil, emperror.Wrap(err, "credentials does not contain a private key")
	}

	o.googleAccessID = jwtConfig.Email
	o.privateKey = jwtConfig.PrivateKey

	ctx := context.Background()

	creds, err := google.CredentialsFromJSON(ctx, credentialsJSON, apiStorage.DevstorageFullControlScope)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get credentials from JSON")
	}

	client, err := storage.NewClient(ctx, option.WithScopes(storage.ScopeReadWrite), option.WithCredentials(creds))
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Google client")
	}
	o.client = client

	return o, nil
}

// CreateBucket creates a new bucket in the object store
func (o *objectStore) CreateBucket(bucketName string) error {
	bucketHandle := o.client.Bucket(bucketName)
	bucketAttrs := &storage.BucketAttrs{
		Location:      o.config.Region,
		RequesterPays: false,
	}

	err := bucketHandle.Create(context.Background(), o.projectID, bucketAttrs)
	if err != nil {
		return emperror.Wrap(o.convertBucketError(err, bucketName), "could not create bucket")
	}

	return nil
}

// ListBuckets lists the current buckets in the object store
func (o *objectStore) ListBuckets() ([]string, error) {
	buckets := make([]string, 0)

	bucketsIterator := o.client.Buckets(context.Background(), o.projectID)

	for {
		bucket, err := bucketsIterator.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, emperror.Wrap(err, "could not list buckets")
		}

		buckets = append(buckets, bucket.Name)
	}

	return buckets, nil
}

// CheckBucket checks the status of the given bucket
func (o *objectStore) CheckBucket(bucketName string) error {
	_, err := o.client.Bucket(bucketName).Attrs(context.Background())
	if err != nil {
		return emperror.Wrap(o.convertBucketError(err, bucketName), "could not check bucket")
	}

	return nil
}

// DeleteBucket deletes a bucket from the object store
func (o *objectStore) DeleteBucket(bucketName string) error {

	obj, err := o.ListObjects(bucketName)
	if err != nil {
		return emperror.WrapWith(err, "failed to list objects", "bucket", bucketName)
	}

	if len(obj) > 0 {
		return emperror.With(pkgErrors.ErrorBucketDeleteNotEmpty, "bucket", bucketName)
	}

	err = o.client.Bucket(bucketName).Delete(context.Background())
	if err != nil {
		return emperror.Wrap(o.convertBucketError(err, bucketName), "could not delete bucket")
	}

	return nil
}

// ListObjects gets all keys in the bucket
func (o *objectStore) ListObjects(bucketName string) ([]string, error) {
	var keys []string

	objects, err := o.listObjectsWithQuery(bucketName, &storage.Query{})
	if err != nil {
		return nil, emperror.Wrap(o.convertBucketError(err, bucketName), "could not list objects")
	}

	for _, object := range objects {
		keys = append(keys, object.Name)
	}

	return keys, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (o *objectStore) ListObjectsWithPrefix(bucketName, prefix string) ([]string, error) {
	var keys []string

	objects, err := o.listObjectsWithQuery(bucketName, &storage.Query{
		Prefix: prefix,
	})
	if err != nil {
		return nil, emperror.WrapWith(o.convertBucketError(err, bucketName), "could not list objects", "prefix", prefix)
	}

	for _, object := range objects {
		keys = append(keys, object.Name)
	}

	return keys, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *objectStore) ListObjectKeyPrefixes(bucketName string, delimiter string) ([]string, error) {
	var prefixes []string

	objects, err := o.listObjectsWithQuery(bucketName, &storage.Query{
		Delimiter: delimiter,
	})
	if err != nil {
		return nil, emperror.WrapWith(o.convertBucketError(err, bucketName), "could not list object key prefixes", "delimeter", delimiter)
	}

	for _, object := range objects {
		if object.Prefix != "" {
			prefixes = append(prefixes, object.Prefix[0:strings.LastIndex(object.Prefix, delimiter)])
		}
	}

	return prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (o *objectStore) GetObject(bucketName string, key string) (io.ReadCloser, error) {
	r, err := o.client.Bucket(bucketName).Object(key).NewReader(context.Background())
	if err != nil {
		return nil, emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not get object")
	}

	return r, nil
}

// PutObject creates a new object using the data in body with the given key
func (o *objectStore) PutObject(bucketName string, key string, body io.Reader) error {
	w := o.client.Bucket(bucketName).Object(key).NewWriter(context.Background())

	_, copyErr := io.Copy(w, body)
	closeErr := w.Close()
	if copyErr != nil {
		return emperror.Wrap(o.convertObjectError(copyErr, bucketName, key), "could not create object")
	}

	if closeErr != nil {
		return emperror.Wrap(o.convertObjectError(closeErr, bucketName, key), "could not create object")
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (o *objectStore) DeleteObject(bucketName string, key string) error {
	err := o.client.Bucket(bucketName).Object(key).Delete(context.Background())
	if err != nil {
		return emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not delete object")
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	url, err := storage.SignedURL(bucketName, key, &storage.SignedURLOptions{
		GoogleAccessID: o.googleAccessID,
		PrivateKey:     o.privateKey,
		Method:         "GET",
		Expires:        time.Now().Add(ttl),
	})
	if err != nil {
		return "", emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not get signed url")
	}

	return url, nil
}

func (o *objectStore) listObjectsWithQuery(bucket string, query *storage.Query) ([]*storage.ObjectAttrs, error) {
	var objects []*storage.ObjectAttrs

	iter := o.client.Bucket(bucket).Objects(context.Background(), query)

	for {
		obj, err := iter.Next()
		if err == iterator.Done {
			return objects, nil
		}
		if err != nil {
			return nil, err
		}

		objects = append(objects, obj)
	}
}

func (o *objectStore) convertBucketError(err error, bucketName string) error {

	if err == storage.ErrBucketNotExist {
		return errBucketNotFound{bucketName: bucketName}
	}

	if gcpErr, ok := err.(*googleapi.Error); ok {
		switch gcpErr.Code {
		case 409:
			return errBucketAlreadyExists{bucketName: bucketName}
		case 404:
			return errBucketNotFound{bucketName: bucketName}
		}
	}

	return emperror.With(err, "bucketName", bucketName)
}

func (o *objectStore) convertObjectError(err error, bucketName, objectName string) error {

	if err == storage.ErrObjectNotExist {
		return errObjectNotFound{bucketName: bucketName, objectName: objectName}
	}

	if gcpErr, ok := err.(*googleapi.Error); ok {
		switch gcpErr.Code {
		case 404:
			return errObjectNotFound{bucketName: bucketName, objectName: objectName}
		}
	}

	return emperror.With(err, "bucket", bucketName, "object", objectName)
}
