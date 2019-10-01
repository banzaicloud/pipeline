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

package securityscan

import (
	"context"

	"github.com/banzaicloud/pipeline/secret"
)

type obj = map[string]interface{}

type secretStoreMock struct {
}

func (s secretStoreMock) Store(ctx context.Context, request *secret.CreateSecretRequest) (string, error) {
	panic("implement me")
}

func (s secretStoreMock) Delete(ctx context.Context, secretID string) error {
	panic("implement me")
}

func (s secretStoreMock) GetSecretValues(ctx context.Context, secretID string) (map[string]string, error) {
	return map[string]string{"username": "test_username", "password": "test_password"}, nil
}
