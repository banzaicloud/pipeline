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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/gin-gonic/gin/json"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	gkeCompute "google.golang.org/api/compute/v1"
	gke "google.golang.org/api/container/v1"
	serviceusagev1beta1 "google.golang.org/api/serviceusage/v1beta1"
)

const (
	ComputeEngineAPI                = "compute.googleapis.com"
	KubernetesEngineAPI             = "container.googleapis.com"
	GoogleCloudStorage              = "storage-component.googleapis.com"
	IAMServiceAccountCredentialsAPI = "iamcredentials.googleapis.com"
	CloudResourceManagerAPI         = "cloudresourcemanager.googleapis.com"
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
	if err != nil {
		return err
	}

	missing, err := checkRequiredServices(client, projectId)
	if err != nil {
		return err
	}
	if missing != nil {
		errorMessage := fmt.Sprintf("required API services are disabled: %s", strings.Join(missing, ","))
		err = errors.New(errorMessage)
	}

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
func createJWTConfig(credentials *ServiceAccount, scope ...string) (*jwt.Config, error) {
	jsonConfig, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	if len(scope) == 0 {
		scope = []string{gke.CloudPlatformScope}
	}
	// Parse credentials from JSON
	return google.JWTConfigFromJSON(jsonConfig, scope...)
}

// CreateOath2Client creates a new OAuth2 client with credentials
func CreateOath2Client(credentials *ServiceAccount, scope ...string) (*http.Client, error) {

	config, err := createJWTConfig(credentials, scope...)
	if err != nil {
		return nil, err
	}

	// Create oauth2 client with credential
	return config.Client(context.TODO()), nil
}

func listServiceUsage(client *http.Client, projectID string) ([]string, error) {
	serviceUsageServiceBeta, err := serviceusagev1beta1.New(client)
	if err != nil {
		return nil, emperror.Wrap(err, "cannot create serviceusage client for checking enabled services")
	}
	enabledServiceCall := serviceUsageServiceBeta.Services.List("projects/" + projectID)
	var enabledServices []string
	nextPageToken := ""
	for {
		resp, err := enabledServiceCall.Context(context.Background()).PageToken(nextPageToken).Do()
		if err != nil {
			return nil, emperror.Wrap(err, "enabled services call failed")
		}
		for _, allServices := range resp.Services {
			if allServices.State == "ENABLED" {
				enabledServices = append(enabledServices, allServices.Name)
			}
		}
		if resp.NextPageToken == "" {
			return enabledServices, nil
		}
		nextPageToken = resp.NextPageToken
	}
}

func checkRequiredServices(client *http.Client, projectID string) ([]string, error) {
	type mustEnabledServices map[string]string
	requiredServices := mustEnabledServices{
		"Compute Engine API":                 ComputeEngineAPI,
		"Kubernetes Engine API":              KubernetesEngineAPI,
		"Google Cloud Storage":               GoogleCloudStorage,
		"IAM ServiceAccount Credentials API": IAMServiceAccountCredentialsAPI,
		"Cloud Resource Manager API":         CloudResourceManagerAPI,
	}

	enabledServices, err := listServiceUsage(client, projectID)
	if err != nil {
		logrus.Error(err)
		return nil, errors.New("list enabled services failed")
	}
	var missingServices []string
Loop:
	for required, value := range requiredServices {
		for _, enabled := range enabledServices {
			if strings.Contains(enabled, value) {
				continue Loop
			}
		}
		missingServices = append(missingServices, required)
	}

	return missingServices, nil
}
