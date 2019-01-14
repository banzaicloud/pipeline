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
	"encoding/json"

	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSecretRequest contains details for a Kubernetes Secret creation from pipeline secrets.
type KubeSecretRequest struct {
	Name      string
	Namespace string
	Type      string
	Values    map[string]string
	Spec      KubeSecretSpec
}

type KubeSecretSpec map[string]KubeSecretSpecItem

type KubeSecretSpecItem struct {
	Source    string
	SourceMap map[string]string
	Value     string
}

// CreateKubeSecret creates a Kubernetes Secret object from a Secret.
func CreateKubeSecret(req KubeSecretRequest) (v1.Secret, error) {
	kubeSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
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

	// Add secret values as is
	if len(req.Spec) == 0 {
		for key, value := range req.Values {
			if opaqueMap[key] {
				continue
			}

			kubeSecret.StringData[key] = value
		}
	} else {
		for key, specItem := range req.Spec {
			if specItem.Source != "" { // Map one secret
				if opaqueMap[specItem.Source] {
					continue
				}

				// TODO: error handling (missing secret key)?
				kubeSecret.StringData[key] = req.Values[specItem.Source]
			} else if specItem.Value == "" { // Map multiple secrets
				sourceMap := make(map[string]string)
				if len(specItem.SourceMap) > 0 { // Map certain secrets
					sourceMap = specItem.SourceMap
				} else { // Include all secrets
					for key := range req.Values {
						sourceMap[key] = key
					}
				}

				data := make(map[string]string)

				for dest, source := range sourceMap {
					if opaqueMap[source] {
						continue
					}

					// TODO: error handling (missing secret key)?
					data[dest] = req.Values[source]
				}

				rawData, err := json.Marshal(data)
				if err != nil {
					return kubeSecret, errors.Wrap(err, "could not marshal secret")
				}

				kubeSecret.StringData[key] = string(rawData)
			} else { // Map value directly
				kubeSecret.StringData[key] = specItem.Value
			}
		}
	}

	return kubeSecret, nil
}
