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

package verify

import (
	"context"
	"net/http"

	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/gin-gonic/gin/json"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	gkeCompute "google.golang.org/api/compute/v1"
	gke "google.golang.org/api/container/v1"
)

// gkeVerify for validation GKE credentials
type gkeVerify struct {
	svc *ServiceAccount
}

// CreateGKESecret create a new 'gkeVerify' instance
func CreateGKESecret(values map[string]string) *gkeVerify {
	return &gkeVerify{
		svc: CreateServiceAccount(values),
	}
}

// VerifySecret validates GKE credentials
func (g *gkeVerify) VerifySecret() error {

	config, err := createJWTConfig(g.svc)
	if err != nil {
		return err
	}

	err = g.refreshToken(config)
	if err != nil {
		return err
	}

	client := config.Client(context.Background())
	return checkProject(client, g.svc.ProjectId)

}

// checkProject validates the project is exists
func checkProject(client *http.Client, projectId string) error {
	service, err := gkeCompute.New(client)
	if err != nil {
		return err
	}

	_, err = getProject(service, projectId)
	return err
}

// getProject returns a project by project id
func getProject(csv *gkeCompute.Service, projectId string) (*gkeCompute.Project, error) {
	return csv.Projects.Get(projectId).Context(context.Background()).Do()
}

// refreshToken returns a token
func (g *gkeVerify) refreshToken(config *jwt.Config) error {
	tokenResource := config.TokenSource(context.Background())
	_, err := tokenResource.Token()
	return err
}

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
		Type:                   values[pkgSecret.Type],
		ProjectId:              values[pkgSecret.ProjectId],
		PrivateKeyId:           values[pkgSecret.PrivateKeyId],
		PrivateKey:             values[pkgSecret.PrivateKey],
		ClientEmail:            values[pkgSecret.ClientEmail],
		ClientId:               values[pkgSecret.ClientId],
		AuthUri:                values[pkgSecret.AuthUri],
		TokenUri:               values[pkgSecret.TokenUri],
		AuthProviderX50CertUrl: values[pkgSecret.AuthX509Url],
		ClientX509CertUrl:      values[pkgSecret.ClientX509Url],
	}
}

// createJWTConfig parses credentials from JSON
func createJWTConfig(credentials *ServiceAccount) (*jwt.Config, error) {
	jsonConfig, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	// Parse credentials from JSON
	return google.JWTConfigFromJSON(jsonConfig, gke.CloudPlatformScope)
}

// CreateOath2Client creates a new OAuth2 client with credentials
func CreateOath2Client(credentials *ServiceAccount) (*http.Client, error) {

	config, err := createJWTConfig(credentials)
	if err != nil {
		return nil, err
	}

	// Create oauth2 client with credential
	return config.Client(context.TODO()), nil
}
