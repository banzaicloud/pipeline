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
	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

type outputSecretInstallManagerS3 struct {
	baseOutputSecretInstallManager
}

func (m outputSecretInstallManagerS3) generateSecretRequest(_ map[string]string, _ bucketSpec) (*pkgCluster.InstallSecretRequest, error) {
	return &pkgCluster.InstallSecretRequest{
		SourceSecretName: m.sourceSecretName,
		Namespace:        m.namespace,
		Spec: map[string]pkgCluster.InstallSecretRequestSpecItem{
			outputDefinitionSecretKeyS3AccessKeyID: {Source: secrettype.AwsAccessKeyId},
			outputDefinitionSecretKeyS3AccessKey:   {Source: secrettype.AwsSecretAccessKey},
		},
		Update: true,
	}, nil
}
