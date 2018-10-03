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

package k8sutil_test

import (
	"testing"

	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergeSecrets(t *testing.T) {
	secret1 := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "secret1",
		},
		StringData: map[string]string{
			"key":  "value1",
			"key1": "value1",
		},
	}

	secret2 := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret2",
			Namespace: "default",
		},
		StringData: map[string]string{
			"key":  "value2",
			"key2": "value2",
		},
	}

	expectedSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret2",
			Namespace: "default",
		},
		StringData: map[string]string{
			"key":  "value2",
			"key1": "value1",
			"key2": "value2",
		},
	}

	mergedSecret := k8sutil.MergeSecrets(secret1, secret2)

	assert.Equal(t, expectedSecret, mergedSecret)
}
