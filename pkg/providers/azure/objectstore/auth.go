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

package objectstore

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
)

// NewClientCredentialsConfigFromSecret returns an Azure client credential config from a secret.
//
// TODO: implement validation for the secret?
func NewClientCredentialsConfigFromSecret(credentials azure.Credentials) auth.ClientCredentialsConfig {
	return auth.NewClientCredentialsConfig(
		credentials.ClientID,
		credentials.ClientSecret,
		credentials.TenantID,
	)
}
