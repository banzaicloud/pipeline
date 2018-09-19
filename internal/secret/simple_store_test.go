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

	. "github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/secret"
)

func TestSimpleStore_Get(t *testing.T) {
	s := &secret.SecretItemResponse{
		Values: map[string]string{
			"key": "value",
		},
	}
	store := NewSimpleStore(&storeMock{
		secret: s,
	})

	secretMap, err := store.Get(1, "secret")
	if err != nil {
		t.Error("unexpected error: ", err)
	}

	if secretMap["key"] != "value" {
		t.Errorf("unexpected secret map: %v", secretMap)
	}
}
