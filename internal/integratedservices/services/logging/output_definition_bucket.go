// Copyright © 2019 Banzai Cloud
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

package logging

import (
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

type bucketOptions struct {
	s3 *struct {
		region string
	}
	gcs *struct {
		project string
	}
}

func generateBucketOptions(spec providerSpec, secretValues map[string]string, orgID uint) (*bucketOptions, error) {
	secretItems := &secret.SecretItemResponse{
		Values: secretValues,
	}
	switch spec.Name {
	case providerAmazonS3:
		return generateS3BucketOptions(spec, secretItems, orgID)
	case providerGoogleGCS:
		return generateGCSBucketOptions(secretValues), nil
	default:
		return &bucketOptions{}, nil
	}
}

func generateS3BucketOptions(spec providerSpec, secretItems *secret.SecretItemResponse, orgID uint) (*bucketOptions, error) {
	region, err := providers.GetBucketLocation(pkgCluster.Amazon, secretItems, spec.Bucket.Name, orgID, nil)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get S3 bucket region", "bucket", spec.Bucket)
	}
	return &bucketOptions{
		s3: &struct {
			region string
		}{
			region: region,
		},
	}, nil
}

func generateGCSBucketOptions(secretValues map[string]string) *bucketOptions {
	return &bucketOptions{
		gcs: &struct {
			project string
		}{
			project: secretValues[secrettype.ProjectId],
		},
	}
}
