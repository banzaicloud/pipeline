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

package secret

import (
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSecretRequest contains details for a Kubernetes Secret creation from pipeline secrets.
type KubeSecretRequest struct {
	Name   string
	Type   string
	Values map[string]string
}

// CreateKubeSecret creates a Kubernetes Secret object from a Secret.
func CreateKubeSecret(req KubeSecretRequest) v1.Secret {
	kubeSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
		StringData: map[string]string{},
	}

	secretMeta := secretTypes.DefaultRules[req.Type]
	opaqueMap := make(map[string]bool, len(secretMeta.Fields))

	// Generic secret fields are never opaque
	if req.Type != secretTypes.GenericSecret {
		for _, field := range secretMeta.Fields {
			opaqueMap[field.Name] = field.Opaque
		}
	}

	for key, value := range req.Values {
		if opaqueMap[key] {
			continue
		}

		kubeSecret.StringData[key] = value
	}

	return kubeSecret
}
