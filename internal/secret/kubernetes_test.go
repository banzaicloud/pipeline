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

package secret_test

import (
	"testing"

	"github.com/banzaicloud/pipeline/internal/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateKubeSecret(t *testing.T) {
	tests := map[string]struct {
		kubeSecret        v1.Secret
		kubeSecretRequest secret.KubeSecretRequest
	}{
		"simple secret": {
			v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				StringData: map[string]string{
					"key": "value",
				},
			},
			secret.KubeSecretRequest{
				Name: "secret",
				Type: "generic",
				Values: map[string]string{
					"key": "value",
				},
			},
		},
		"secret with opaque fields": {
			v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				StringData: map[string]string{
					".htpasswd": "blah",
				},
			},
			secret.KubeSecretRequest{
				Name: "secret",
				Type: "htpasswd",
				Values: map[string]string{
					"username":  "user",
					"password":  "pass",
					".htpasswd": "blah",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			kubeSecret, err := secret.CreateKubeSecret(test.kubeSecretRequest)
			require.NoError(t, err)

			assert.Equal(t, test.kubeSecret, kubeSecret)
		})
	}
}
