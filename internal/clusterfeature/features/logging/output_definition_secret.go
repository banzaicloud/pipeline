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

	pkgCluster "github.com/banzaicloud/pipeline/src/cluster"
)

type outputSecretInstallManager interface {
	generateSecretRequest(map[string]string, bucketSpec) (*pkgCluster.InstallSecretRequest, error)
}

type baseOutputSecretInstallManager struct {
	sourceSecretName string
	namespace        string
}

func newOutputSecretInstallManager(providerName, sourceSecretName, namespace string) (outputSecretInstallManager, error) {
	switch providerName {
	case providerAmazonS3:
		return outputSecretInstallManagerS3{baseOutputSecretInstallManager{
			sourceSecretName: sourceSecretName,
			namespace:        namespace,
		}}, nil
	case providerGoogleGCS:
		return outputSecretInstallManagerGCS{baseOutputSecretInstallManager{
			sourceSecretName: sourceSecretName,
			namespace:        namespace,
		}}, nil
	case providerAzure:
		return outputSecretInstallManagerAzure{baseOutputSecretInstallManager{
			sourceSecretName: sourceSecretName,
			namespace:        namespace,
		}}, nil
	case providerAlibabaOSS:
		return outputSecretInstallManagerOSS{baseOutputSecretInstallManager{
			sourceSecretName: sourceSecretName,
			namespace:        namespace,
		}}, nil
	default:
		return nil, errors.NewWithDetails("unsupported provider", "provider", providerName)
	}
}
