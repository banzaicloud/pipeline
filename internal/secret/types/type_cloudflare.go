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
	"github.com/banzaicloud/pipeline/internal/secret"
)

const Cloudflare = "cloudflare"

const (
	FieldCloudflareApiKey   = "CF_API_KEY"
	FieldCloudflareApiEmail = "CF_API_EMAIL"
)

type CloudflareType struct{}

func (CloudflareType) Name() string {
	return Cloudflare
}

func (CloudflareType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldCloudflareApiKey, Required: true, Opaque: true, Description: "Your API key"},
			{Name: FieldCloudflareApiEmail, Required: true, IsSafeToDisplay: true, Opaque: true, Description: "Your API E-mail"},
		},
	}
}

func (t CloudflareType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}
