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
	"time"

	"github.com/banzaicloud/pipeline/internal/secret"
)

// DefaultTypeListConfig contains the required configuration for the default type list.
type DefaultTypeListConfig struct {
	AmazonRegion       string
	TLSDefaultValidity time.Duration
	PkeSecreter        PkeSecreter
}

// NewDefaultTypeList returns a TypeList with all default types.
func NewDefaultTypeList(config DefaultTypeListConfig) secret.TypeList {
	return secret.NewTypeList([]secret.Type{
		AmazonType{Region: config.AmazonRegion},
		AzureType{},
		AzureStorageAccountType{},
		CloudflareType{},
		DigitalOceanType{},
		FnType{},
		GenericType{},
		GoogleType{},
		HtpasswdType{},
		KubernetesType{},
		PagerDutyType{},
		PasswordType{},
		PKEType{PkeSecreter: config.PkeSecreter},
		SlackType{},
		SSHType{},
		TLSType{DefaultValidity: config.TLSDefaultValidity},
		VaultType{},
		VsphereType{},
	})
}
