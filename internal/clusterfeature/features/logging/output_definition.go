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
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	"github.com/banzaicloud/logging-operator/pkg/sdk/model/output"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/secret"
)

type baseOutputManager struct {
	sourceSecretName string
}

type outputDefinitionManager interface {
	getOutputSpec(clusterOutputSpec, bucketOptions) v1beta1.ClusterOutputSpec
	getOutputName() string
	getFlowName() string
}

func newOutputDefinitionManager(providerName, sourceSecretName string) (outputDefinitionManager, error) {
	switch providerName {
	case providerAmazonS3:
		return outputDefinitionManagerS3{baseOutputManager{sourceSecretName: sourceSecretName}}, nil
	case providerGoogleGCS:
		return outputDefinitionManagerGCS{baseOutputManager{sourceSecretName: sourceSecretName}}, nil
	case providerAzure:
		return outputDefinitionManagerAzure{baseOutputManager{sourceSecretName: sourceSecretName}}, nil
	case providerAlibabaOSS:
		return outputDefinitionManagerOSS{baseOutputManager{sourceSecretName: sourceSecretName}}, nil
	default:
		return nil, errors.NewWithDetails("unsupported provider", "provider", providerName)
	}
}

type bucketOptions struct {
	s3 *struct {
		region string
	}
	oss *struct {
		region string
	}
	gcs *struct {
		project string
	}
}

func (baseOutputManager) getBufferSpec() *output.Buffer {
	return &output.Buffer{
		Timekey:       "1m",
		TimekeyWait:   "10s",
		TimekeyUseUtc: true,
	}
}

func (baseOutputManager) getPathSpec() string {
	return "logs/${tag}/%Y/%m/%d/"
}

func generateOutputDefinition(
	ctx context.Context,
	m outputDefinitionManager,
	secretStore common.SecretStore,
	spec clusterOutputSpec,
	namespace string,
	orgID uint,
) (*v1beta1.ClusterOutput, error) {
	secretValues, err := secretStore.GetSecretValues(ctx, spec.Provider.SecretID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get secret", "secretID", spec.Provider.SecretID)
	}

	bucketOptions, err := generateBucketOptions(spec.Provider, secretValues, orgID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to generate bucket options")
	}

	return &v1beta1.ClusterOutput{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getOutputName(),
			Namespace: namespace,
		},
		Spec: m.getOutputSpec(spec, *bucketOptions),
	}, nil
}

func generateBucketOptions(spec providerSpec, secretValues map[string]string, orgID uint) (*bucketOptions, error) {
	var secretItems = &secret.SecretItemResponse{
		Values: secretValues,
	}
	switch spec.Name {
	case providerAmazonS3:
		return generateS3BucketOptions(spec, secretItems, orgID)
	case providerGoogleGCS:
		return generateGCSBucketOptions(secretValues), nil
	case providerAlibabaOSS:
		return generateOSSBucketOptions(spec, secretItems, orgID)
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

func generateOSSBucketOptions(spec providerSpec, secretItems *secret.SecretItemResponse, orgID uint) (*bucketOptions, error) {
	region, err := providers.GetBucketLocation(pkgCluster.Alibaba, secretItems, spec.Bucket.Name, orgID, nil)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get OSS bucket region", "bucket", spec.Bucket.Name)
	}
	return &bucketOptions{
		oss: &struct {
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
