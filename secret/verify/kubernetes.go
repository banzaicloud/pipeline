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

package verify

import (
	"encoding/base64"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

type kubernetesConfigVerifier struct {
	kubeConfigStr string
}

func (v *kubernetesConfigVerifier) VerifySecret() error {
	if v.kubeConfigStr == "" {
		return errors.New("no Kubernetes config provided")
	}

	kubeConfig, err := base64.StdEncoding.DecodeString(v.kubeConfigStr)
	if err != nil {
		return errors.WrapIf(err, "can't decode Kubernetes config")
	}

	clientSet, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "couldn't get Kubernetes client")
	}

	_, err = clientSet.ServerVersion()

	return errors.WrapIf(err, "couldn't validate Kubernetes config")
}

// CreateKubeConfigSecretVerifier creates a verifier that checks if the provided KubeConfig
// is valid by trying to execute simple operation (get api server version) using it
func CreateKubeConfigSecretVerifier(values map[string]string) *kubernetesConfigVerifier {
	return &kubernetesConfigVerifier{
		kubeConfigStr: values[secrettype.K8SConfig],
	}
}
