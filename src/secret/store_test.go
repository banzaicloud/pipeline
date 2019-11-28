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
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

func TestGetValue(t *testing.T) {

	cases := []struct {
		name          string
		secretItem    secret.SecretItemResponse
		searchedKey   string
		expectedValue string
	}{
		{name: "gke project id", secretItem: secretItem1, searchedKey: secrettype.ProjectId, expectedValue: gkeProjectId},
		{name: "non", secretItem: secretItem1, searchedKey: secretProjectId2, expectedValue: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := tc.secretItem.GetValue(tc.searchedKey)
			if !reflect.DeepEqual(item, tc.expectedValue) {
				t.Errorf("Expected value: %s, but got: %s", tc.expectedValue, item)
			}
		})
	}

}

func TestCreateSecretValidate(t *testing.T) {

	cases := []struct {
		name     string
		request  secret.CreateSecretRequest
		isError  bool
		verifier verify.Verifier
	}{
		{name: "aws full", request: awsCreateSecretFull, isError: false, verifier: nil},
		{name: "aks full", request: aksCreateSecretFull, isError: false, verifier: nil},
		{name: "gke full", request: gkeCreateSecretFull, isError: false, verifier: nil},
		{name: "ssh full", request: sshCreateSecretFull, isError: false, verifier: nil},
		{name: "oracle full", request: okeCreateSecretFull, isError: false, verifier: nil},

		{name: "aws missing key", request: awsMissingKey, isError: true, verifier: nil},
		{name: "aks missing key", request: aksMissingKey, isError: true, verifier: nil},
		{name: "gke missing key", request: gkeMissingKey, isError: true, verifier: nil},
		{name: "ssh missing key", request: sshMissingKey, isError: true, verifier: nil},
		{name: "oracle missing key", request: okeMissingKey, isError: true, verifier: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate(tc.verifier)

			if err != nil {
				if !tc.isError {
					t.Errorf("Error occours: %s", err.Error())
				}
			} else if tc.isError {
				t.Errorf("Not occours error")
			}

		})
	}

}

const (
	secretId         = "secretId"
	secretDesc       = "secretDesc"
	secretProjectId2 = "testProjectId2"
)

const (
	AwsAccessKeyId     = "accessKey"
	AwsSecretAccessKey = "secretKey"
)

const (
	AzureClientID       = "AZURE_CLIENT_ID"
	AzureClientSecret   = "AZURE_CLIENT_SECRET"
	AzureTenantID       = "AZURE_TENANT_ID"
	AzureSubscriptionID = "AZURE_SUBSCRIPTION_ID"
)

const (
	gkeType         = "testType"
	gkeProjectId    = "project_id"
	gkePrivateKeyId = "private_key_id"
	gkePrivateKey   = "private_key"
	gkeClientEmail  = "client_email"
	gkeClientId     = "client_id"
	gkeAuthUri      = "auth_uri"
	gkeTokenUri     = "token_uri"
	gkeAuthCert     = "auth_provider_x509_cert_url"
	gkeClientCert   = "client_x509_cert_url"
)

const (
	SshUser                 = "user"
	SshIdentifier           = "identifier"
	SshPublicKeyData        = "public_key_data"
	SshPublicKeyFingerprint = "public_key_fingerprint"
	SshPrivateKeyData       = "private_key_data"
)

// nolint: gochecknoglobals
var (
	awsCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Amazon,
		Values: map[string]string{
			secrettype.AwsAccessKeyId:     AwsAccessKeyId,
			secrettype.AwsSecretAccessKey: AwsSecretAccessKey,
		},
	}

	awsMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Amazon,
		Values: map[string]string{
			secrettype.AwsSecretAccessKey: AwsSecretAccessKey,
		},
	}

	aksCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Azure,
		Values: map[string]string{
			secrettype.AzureClientID:       AzureClientID,
			secrettype.AzureClientSecret:   AzureClientSecret,
			secrettype.AzureTenantID:       AzureTenantID,
			secrettype.AzureSubscriptionID: AzureSubscriptionID,
		},
	}

	aksMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Azure,
		Values: map[string]string{
			secrettype.AzureClientID:       AzureClientID,
			secrettype.AzureSubscriptionID: AzureSubscriptionID,
		},
	}

	gkeCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Google,
		Values: map[string]string{
			secrettype.Type:          gkeType,
			secrettype.ProjectId:     gkeProjectId,
			secrettype.PrivateKeyId:  gkePrivateKeyId,
			secrettype.PrivateKey:    gkePrivateKey,
			secrettype.ClientEmail:   gkeClientEmail,
			secrettype.ClientId:      gkeClientId,
			secrettype.AuthUri:       gkeAuthUri,
			secrettype.TokenUri:      gkeTokenUri,
			secrettype.AuthX509Url:   gkeAuthCert,
			secrettype.ClientX509Url: gkeClientCert,
		},
	}

	gkeMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Google,
		Values: map[string]string{
			secrettype.Type:          gkeType,
			secrettype.ProjectId:     gkeProjectId,
			secrettype.PrivateKeyId:  gkePrivateKeyId,
			secrettype.PrivateKey:    gkePrivateKey,
			secrettype.ClientId:      gkeClientId,
			secrettype.AuthUri:       gkeAuthUri,
			secrettype.TokenUri:      gkeTokenUri,
			secrettype.AuthX509Url:   gkeAuthCert,
			secrettype.ClientX509Url: gkeClientCert,
		},
	}

	sshCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: secrettype.SSHSecretType,
		Values: map[string]string{
			secrettype.User:                 SshUser,
			secrettype.Identifier:           SshIdentifier,
			secrettype.PublicKeyData:        SshPublicKeyData,
			secrettype.PublicKeyFingerprint: SshPublicKeyFingerprint,
			secrettype.PrivateKeyData:       SshPrivateKeyData,
		},
	}

	sshMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: secrettype.SSHSecretType,
		Values: map[string]string{
			secrettype.User:                 SshUser,
			secrettype.Identifier:           SshIdentifier,
			secrettype.PublicKeyData:        SshPublicKeyData,
			secrettype.PublicKeyFingerprint: SshPublicKeyFingerprint,
		},
	}

	okeCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Oracle,
		Values: map[string]string{
			secrettype.OracleUserOCID:          secrettype.OracleUserOCID,
			secrettype.OracleTenancyOCID:       secrettype.OracleTenancyOCID,
			secrettype.OracleAPIKey:            secrettype.OracleAPIKey,
			secrettype.OracleAPIKeyFingerprint: secrettype.OracleAPIKeyFingerprint,
			secrettype.OracleRegion:            secrettype.OracleRegion,
			secrettype.OracleCompartmentOCID:   secrettype.OracleCompartmentOCID,
		},
	}

	okeMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pkgCluster.Oracle,
		Values: map[string]string{
			secrettype.OracleUserOCID:          secrettype.OracleUserOCID,
			secrettype.OracleTenancyOCID:       secrettype.OracleTenancyOCID,
			secrettype.OracleAPIKey:            secrettype.OracleAPIKey,
			secrettype.OracleAPIKeyFingerprint: secrettype.OracleAPIKeyFingerprint,
			secrettype.OracleRegion:            secrettype.OracleRegion,
		},
	}
)

// nolint: gochecknoglobals
var (
	secretItem1 = secret.SecretItemResponse{
		ID:   secretId,
		Name: secretDesc,
		Type: pkgCluster.Google,
		Values: map[string]string{
			secrettype.ProjectId: gkeProjectId,
		},
	}
)
