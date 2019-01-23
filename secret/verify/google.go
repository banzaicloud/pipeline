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
	"strconv"
	"strings"

	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/gin-gonic/gin/json"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/serviceusage/v1"
)

const (
	ComputeEngineAPI                = "compute.googleapis.com"
	KubernetesEngineAPI             = "container.googleapis.com"
	GoogleCloudStorage              = "storage-component.googleapis.com"
	IAMServiceAccountCredentialsAPI = "iamcredentials.googleapis.com"
	CloudResourceManagerAPI         = "cloudresourcemanager.googleapis.com"
)

// GCPSecretVerifier represents a secret verifier for Google Cloud Platform secrets
type GCPSecretVerifier struct {
	*ServiceAccount
}

// CreateGCPSecretVerifier creates a new Google Cloud Platform secret verifier
func CreateGCPSecretVerifier(values map[string]string) GCPSecretVerifier {
	return GCPSecretVerifier{CreateServiceAccount(values)}
}

// VerifySecret validates GCP credentials
func (sv GCPSecretVerifier) VerifySecret() error {
	return checkProject(sv.ServiceAccount)
}

func checkProject(serviceAccount *ServiceAccount) error {
	missing, err := checkRequiredServices(serviceAccount)
	if err != nil {
		return err
	}
	if len(missing) != 0 {
		return fmt.Errorf("required API services are disabled: %s", strings.Join(missing, ","))
	}
	return nil
}

func checkRequiredServices(serviceAccount *ServiceAccount) ([]string, error) {
	requiredServices := map[string]string{
		ComputeEngineAPI:                "Compute Engine API",
		KubernetesEngineAPI:             "Kubernetes Engine API",
		GoogleCloudStorage:              "Google Cloud Storage",
		IAMServiceAccountCredentialsAPI: "IAM ServiceAccount Credentials API",
		CloudResourceManagerAPI:         "Cloud Resource Manager API",
	}

	enabledServices, err := listEnabledServices(serviceAccount)
	if err != nil {
		logrus.Error(err)
		return nil, errors.New("list enabled services failed")
	}

	var missingServices []string
	for service, readableName := range requiredServices {
		if !contains(enabledServices, service) {
			missingServices = append(missingServices, readableName)
		}
	}
	return missingServices, nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func listEnabledServices(serviceAccount *ServiceAccount) ([]string, error) {
	client, err := CreateOath2Client(serviceAccount, serviceusage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	crmSvc, err := cloudresourcemanager.New(client)
	if err != nil {
		return nil, err
	}
	project, err := crmSvc.Projects.Get(serviceAccount.ProjectId).Do()
	if err != nil {
		return nil, err
	}
	suSvc, err := serviceusage.New(client)
	if err != nil {
		return nil, emperror.Wrap(err, "cannot create serviceusage client for checking enabled services")
	}
	enabledServicesCall := suSvc.Services.List("projects/" + strconv.FormatInt(project.ProjectNumber, 10)).Filter("state:ENABLED").Fields("services/config/name")

	var enabledServices []string
	nextPageToken := ""
	for {
		resp, err := enabledServicesCall.PageToken(nextPageToken).Do()
		if err != nil {
			return nil, emperror.Wrap(err, "enabled services call failed")
		}
		for _, service := range resp.Services {
			enabledServices = append(enabledServices, service.Config.Name)
		}
		if resp.NextPageToken == "" {
			return enabledServices, nil
		}
		nextPageToken = resp.NextPageToken
	}
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
