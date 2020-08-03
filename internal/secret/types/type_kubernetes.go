// Copyright © 2020 Banzai Cloud
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

package types

import (
	"encoding/base64"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const Kubernetes = "kubernetes"

const (
	FieldKubernetesConfig = "K8Sconfig"
)

type KubernetesType struct{}

func (KubernetesType) Name() string {
	return Kubernetes
}

func (KubernetesType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldKubernetesConfig, Required: true},
		},
	}
}

func (t KubernetesType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}

func (t KubernetesType) Process(data map[string]string) (map[string]string, error) {
	if _, err := base64.StdEncoding.DecodeString(data[FieldKubernetesConfig]); err != nil {
		data[FieldKubernetesConfig] = base64.StdEncoding.EncodeToString([]byte(data[FieldKubernetesConfig]))
	}

	return data, nil
}

// TODO: rewrite this function!
func (KubernetesType) Verify(data map[string]string) error {
	err := kubernetesVerify(data)
	if err != nil {
		return secret.NewValidationError(err.Error(), nil)
	}

	return nil
}

func kubernetesVerify(data map[string]string) error {
	if data[FieldKubernetesConfig] == "" {
		return errors.New("no Kubernetes config provided")
	}

	kubeConfig, err := base64.StdEncoding.DecodeString(data[FieldKubernetesConfig])
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
