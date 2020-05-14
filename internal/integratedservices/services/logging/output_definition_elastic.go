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
	"github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	"github.com/banzaicloud/logging-operator/pkg/sdk/model/output"
	loggingSecret "github.com/banzaicloud/logging-operator/pkg/sdk/model/secret"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
)

type outputDefinitionManagerElastic struct {
	baseOutputManager

	secret secret.SecretItemResponse
}

func (o outputDefinitionManagerElastic) getOutputSpec(_ bucketSpec, _ bucketOptions) v1beta1.ClusterOutputSpec {
	return v1beta1.ClusterOutputSpec{
		OutputSpec: v1beta1.OutputSpec{
			ElasticsearchOutput: &output.ElasticsearchOutput{
				Host:       "localhost", // todo (colin)?? host
				Port:       9200,
				Scheme:     "https",
				SslVerify:  false,
				SslVersion: "TLSv1_2",
				User:       o.secret.Values[secrettype.Username],
				Password: &loggingSecret.Secret{
					ValueFrom: &loggingSecret.ValueFrom{
						SecretKeyRef: &loggingSecret.KubernetesSecret{
							Name: o.sourceSecretName,
							Key:  outputDefinitionSecretKeyElasticSearch,
						},
					},
				},
				Path:   o.getPathSpec(),
				Buffer: o.getBufferSpec(),
			},
		},
	}
}

func (outputDefinitionManagerElastic) getName() string {
	return elasticOutputDefinitionName
}

func (outputDefinitionManagerElastic) getProviderSpec() providerSpec {
	return providerSpec{}
}
