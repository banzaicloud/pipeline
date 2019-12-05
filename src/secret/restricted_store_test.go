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
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
)

const (
	orgID = 19
)

// nolint: gochecknoglobals
var version = 1

func TestBlockingTags(t *testing.T) {

	cases := []struct {
		name    string
		request *secret.CreateSecretRequest
	}{
		{name: "readonly", request: &requestReadOnly},
		{name: "forbidden", request: &requestForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			secretID, err := secret.RestrictedStore.Store(orgID, tc.request)

			if err != nil {
				t.Errorf("error during storing readonly secret: %s", err.Error())
				t.FailNow()
			}

			err = secret.RestrictedStore.Delete(orgID, secretID)
			if err == nil {
				t.Error("readonly secret deleted..")
				t.FailNow()

				tc.request.Tags = append(tc.request.Tags, "newtag")

				err = secret.RestrictedStore.Update(orgID, secretID, tc.request)
				if err == nil {
					t.Error("readonly secret updated..")
					t.FailNow()
				}

				expErr := secret.ReadOnlyError{SecretID: secretID}
				if !reflect.DeepEqual(err, expErr) {
					t.Errorf("expected error: %s, got: %s", expErr, err.Error())
					t.FailNow()
				}
			}
		})
	}

}

// nolint: gochecknoglobals
var (
	requestReadOnly = secret.CreateSecretRequest{
		Name: "readonly",
		Type: secrettype.Password,
		Values: map[string]string{
			"key": "value",
		},
		Tags: []string{
			secret.TagBanzaiReadonly,
		},
		Version:   version,
		UpdatedBy: "banzaiuser",
	}

	requestForbidden = secret.CreateSecretRequest{
		Name: "forbidden",
		Type: secrettype.Password,
		Values: map[string]string{
			"key": "value",
		},
		Tags:      secret.ForbiddenTags,
		Version:   version,
		UpdatedBy: "banzaiuser",
	}
)
