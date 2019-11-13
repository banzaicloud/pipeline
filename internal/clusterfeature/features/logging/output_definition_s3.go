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
)

type outputDefinitionManagerS3 struct {
	baseOutputManager
}

func (outputDefinitionManagerS3) getName() string {
	return "s3-output"
}

func (m outputDefinitionManagerS3) getOutputSpec(spec clusterOutputSpec, op bucketOptions) v1beta1.OutputSpec {
	return v1beta1.OutputSpec{
		S3OutputConfig: &output.S3OutputConfig{
			AwsAccessKey: &loggingSecret.Secret{
				ValueFrom: &loggingSecret.ValueFrom{
					SecretKeyRef: &loggingSecret.KubernetesSecret{
						Name: m.sourceSecretName,
						Key:  outputDefinitionSecretKeyS3AccessKeyID,
					},
				},
			},
			AwsSecretKey: &loggingSecret.Secret{
				ValueFrom: &loggingSecret.ValueFrom{
					SecretKeyRef: &loggingSecret.KubernetesSecret{
						Name: m.sourceSecretName,
						Key:  outputDefinitionSecretKeyS3AccessKey,
					},
				},
			},
			Path:     m.getPathSpec(),
			S3Region: op.s3.region,
			S3Bucket: spec.Provider.Bucket.Name,
			Buffer:   m.getBufferSpec(),
			Format: &output.Format{
				Type: "json",
			},
		},
	}
}
