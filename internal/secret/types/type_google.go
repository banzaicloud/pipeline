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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/serviceusage/v1"

	"github.com/banzaicloud/pipeline/internal/secret"
)

const Google = "google"

const (
	FieldGoogleType          = "type"
	FieldGoogleProjectId     = "project_id"
	FieldGooglePrivateKeyId  = "private_key_id"
	FieldGooglePrivateKey    = "private_key"
	FieldGoogleClientEmail   = "client_email"
	FieldGoogleClientId      = "client_id"
	FieldGoogleAuthUri       = "auth_uri"
	FieldGoogleTokenUri      = "token_uri"
	FieldGoogleAuthX509Url   = "auth_provider_x509_cert_url"
	FieldGoogleClientX509Url = "client_x509_cert_url"
)

type GoogleType struct{}

func (GoogleType) Name() string {
	return Google
}

func (GoogleType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldGoogleType, Required: true, IsSafeToDisplay: true, Description: "service_account"},
			{Name: FieldGoogleProjectId, Required: true, IsSafeToDisplay: true, Description: "Google Could Project Id. Find more about, Google Cloud secret fields here: https://banzaicloud.com/docs/pipeline/secrets/providers/gke_auth_credentials/#method-2-command-line"},
			{Name: FieldGooglePrivateKeyId, Required: true, IsSafeToDisplay: true, Description: "Id of you private key"},
			{Name: FieldGooglePrivateKey, Required: true, Description: "Your private key "},
			{Name: FieldGoogleClientEmail, Required: true, IsSafeToDisplay: true, Description: "Google service account client email"},
			{Name: FieldGoogleClientId, Required: true, IsSafeToDisplay: true, Description: "Client Id"},
			{Name: FieldGoogleAuthUri, Required: true, IsSafeToDisplay: true, Description: "OAuth2 authentatication IRU"},
			{Name: FieldGoogleTokenUri, Required: true, IsSafeToDisplay: true, Description: "OAuth2 token URI"},
			{Name: FieldGoogleAuthX509Url, Required: true, IsSafeToDisplay: true, Description: "OAuth2 provider ceritficate URL"},
			{Name: FieldGoogleClientX509Url, Required: true, IsSafeToDisplay: true, Description: "OAuth2 client ceritficate URL"},
		},
	}
}

func (t GoogleType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}

// TODO: rewrite this function!
func (GoogleType) Verify(data map[string]string) error {
	err := googleCheckProject(googleCreateServiceAccount(data))
	if err != nil {
		return secret.NewValidationError(err.Error(), nil)
	}

	return nil
}

const (
	googlePermissionComputeEngineAPI                = "compute.googleapis.com"
	googlePermissionKubernetesEngineAPI             = "container.googleapis.com"
	googlePermissionCloudStorage                    = "storage-component.googleapis.com"
	googlePermissionIAMServiceAccountCredentialsAPI = "iamcredentials.googleapis.com"
	googlePermissionCloudResourceManagerAPI         = "cloudresourcemanager.googleapis.com"
)

// googleServiceAccount describes a GKE service account
type googleServiceAccount struct {
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

func googleCreateServiceAccount(values map[string]string) *googleServiceAccount {
	return &googleServiceAccount{
		Type:                   values[FieldGoogleType],
		ProjectId:              values[FieldGoogleProjectId],
		PrivateKeyId:           values[FieldGooglePrivateKeyId],
		PrivateKey:             values[FieldGooglePrivateKey],
		ClientEmail:            values[FieldGoogleClientEmail],
		ClientId:               values[FieldGoogleClientId],
		AuthUri:                values[FieldGoogleAuthUri],
		TokenUri:               values[FieldGoogleTokenUri],
		AuthProviderX50CertUrl: values[FieldGoogleAuthX509Url],
		ClientX509CertUrl:      values[FieldGoogleClientX509Url],
	}
}

func googleCheckProject(serviceAccount *googleServiceAccount) error {
	missing, err := googleCheckRequiredServices(serviceAccount)
	if err != nil {
		return err
	}

	if len(missing) != 0 {
		return fmt.Errorf("required API services are disabled: %s", strings.Join(missing, ","))
	}

	return nil
}

func googleCheckRequiredServices(serviceAccount *googleServiceAccount) ([]string, error) {
	requiredServices := map[string]string{
		googlePermissionComputeEngineAPI:                "Compute Engine API",
		googlePermissionKubernetesEngineAPI:             "Kubernetes Engine API",
		googlePermissionCloudStorage:                    "Google Cloud Storage",
		googlePermissionIAMServiceAccountCredentialsAPI: "IAM ServiceAccount Credentials API",
		googlePermissionCloudResourceManagerAPI:         "Cloud Resource Manager API",
	}

	enabledServices, err := googleListEnabledServices(serviceAccount)
	if err != nil {
		return nil, errors.WrapIf(err, "list enabled services failed")
	}

	var missingServices []string
	for service, readableName := range requiredServices {
		if !googleContains(enabledServices, service) {
			missingServices = append(missingServices, readableName)
		}
	}
	return missingServices, nil
}

func googleContains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func googleListEnabledServices(serviceAccount *googleServiceAccount) ([]string, error) {
	client, err := googleCreateOath2Client(serviceAccount, serviceusage.CloudPlatformScope)
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
		return nil, errors.WrapIf(err, "cannot create serviceusage client for checking enabled services")
	}
	enabledServicesCall := suSvc.Services.List("projects/" + strconv.FormatInt(project.ProjectNumber, 10)).Filter("state:ENABLED").Fields("services/config/name")

	var enabledServices []string
	nextPageToken := ""
	for {
		resp, err := enabledServicesCall.PageToken(nextPageToken).Do()
		if err != nil {
			return nil, errors.WrapIf(err, "enabled services call failed")
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

// googleCreateJWTConfig parses credentials from JSON
func googleCreateJWTConfig(credentials *googleServiceAccount, scope ...string) (*jwt.Config, error) {
	jsonConfig, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	return google.JWTConfigFromJSON(jsonConfig, scope...)
}

// googleCreateOath2Client creates a new OAuth2 client with credentials
func googleCreateOath2Client(serviceAccount *googleServiceAccount, scope ...string) (*http.Client, error) {
	if len(scope) == 0 {
		// This is here for backward compatibility, but it should probably be explicitly stated everywhere
		scope = []string{serviceusage.CloudPlatformScope}
	}
	config, err := googleCreateJWTConfig(serviceAccount, scope...)
	if err != nil {
		return nil, err
	}
	return config.Client(context.Background()), nil
}
