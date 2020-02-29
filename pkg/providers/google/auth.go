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

package google

import (
	"context"
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/serviceusage/v1"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// ServiceAccount describes a GKE service account
type ServiceAccount struct {
	Type                   string `json:"type"`
	ProjectId              string `json:"project_id"`
	PrivateKeyId           string `json:"private_key_id"`
	PrivateKey             string `json:"private_key"`
	ClientEmail            string `json:"client_email"`
	ClientId               string `json:"client_id"`
	AuthUri                string `json:"auth_uri"`
	TokenUri               string `json:"token_uri"`
	AuthProviderX50CertUrl string `json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl      string `json:"client_x509_cert_url"`
}

// CreateServiceAccount creates a new 'ServiceAccount' instance
func CreateServiceAccount(values map[string]string) *ServiceAccount {
	return &ServiceAccount{
		Type:                   values[secrettype.Type],
		ProjectId:              values[secrettype.ProjectId],
		PrivateKeyId:           values[secrettype.PrivateKeyId],
		PrivateKey:             values[secrettype.PrivateKey],
		ClientEmail:            values[secrettype.ClientEmail],
		ClientId:               values[secrettype.ClientId],
		AuthUri:                values[secrettype.AuthUri],
		TokenUri:               values[secrettype.TokenUri],
		AuthProviderX50CertUrl: values[secrettype.AuthX509Url],
		ClientX509CertUrl:      values[secrettype.ClientX509Url],
	}
}

// createJWTConfig parses credentials from JSON
func createJWTConfig(credentials *ServiceAccount, scope ...string) (*jwt.Config, error) {
	jsonConfig, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	return google.JWTConfigFromJSON(jsonConfig, scope...)
}

// CreateOath2Client creates a new OAuth2 client with credentials
func CreateOath2Client(serviceAccount *ServiceAccount, scope ...string) (*http.Client, error) {
	if len(scope) == 0 {
		// This is here for backward compatibility, but it should probably be explicitly stated everywhere
		scope = []string{serviceusage.CloudPlatformScope}
	}
	config, err := createJWTConfig(serviceAccount, scope...)
	if err != nil {
		return nil, err
	}
	return config.Client(context.Background()), nil
}
