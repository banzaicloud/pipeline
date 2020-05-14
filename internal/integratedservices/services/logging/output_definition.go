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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/common"
)

type outputManagerCreator struct {
	name             string
	sourceSecretName string
	serviceURL       string
	providerSpec     providerSpec
}

type outputDefinitionManager interface {
	getOutputSpec(bucketSpec, bucketOptions) v1beta1.ClusterOutputSpec
	getProviderSpec() providerSpec
	getName() string
}

func newOutputDefinitionManager(creators []outputManagerCreator) (managers []outputDefinitionManager) {
	for _, creator := range creators {
		var baseManager = baseOutputManager{
			sourceSecretName: creator.sourceSecretName,
			providerSpec:     creator.providerSpec,
		}
		switch creator.name {
		case providerAmazonS3:
			managers = append(managers, outputDefinitionManagerS3{baseOutputManager: baseManager})
		case providerGoogleGCS:
			managers = append(managers, outputDefinitionManagerGCS{baseOutputManager: baseManager})
		case providerAzure:
			managers = append(managers, outputDefinitionManagerAzure{baseOutputManager: baseManager})
		case providerAlibabaOSS:
			managers = append(managers, outputDefinitionManagerOSS{baseOutputManager: baseManager})
		case providerLoki:
			managers = append(managers, outputDefinitionManagerLoki{serviceURL: creator.serviceURL})
		case providerElasticSearch:
			managers = append(managers, outputDefinitionManagerElastic{})
		}
	}

	return
}

func generateOutputDefinition(
	ctx context.Context,
	m outputDefinitionManager,
	secretStore common.SecretStore,
	namespace string,
	orgID uint,
) (*v1beta1.ClusterOutput, error) {
	var spec = m.getProviderSpec()
	var bucketOptions = &bucketOptions{}
	if spec.SecretID != "" {
		secretValues, err := secretStore.GetSecretValues(ctx, spec.SecretID)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to get secret", "secretID", spec.SecretID)
		}

		bucketOptions, err = generateBucketOptions(spec, secretValues, orgID)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to generate bucket options")
		}
	}

	return &v1beta1.ClusterOutput{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getName(),
			Namespace: namespace,
			Labels:    map[string]string{resourceLabelKey: integratedServiceName},
		},
		Spec: m.getOutputSpec(spec.Bucket, *bucketOptions),
	}, nil
}
