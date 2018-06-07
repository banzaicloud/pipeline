package secret_test

import (
	"reflect"
	"testing"

	btypes "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth/cloud"
	pipConstants "github.com/banzaicloud/pipeline/constants"
	"github.com/banzaicloud/pipeline/secret"
)

func TestGetValue(t *testing.T) {

	cases := []struct {
		name          string
		secretItem    secret.SecretsItemResponse
		searchedKey   string
		expectedValue string
	}{
		{name: "gke project id", secretItem: secretItem1, searchedKey: pipConstants.ProjectId, expectedValue: gkeProjectId},
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
		verifier cloud.Verifier
	}{
		{name: "aws full", request: awsCreateSecretFull, isError: false, verifier: nil},
		{name: "aks full", request: aksCreateSecretFull, isError: false, verifier: nil},
		{name: "gke full", request: gkeCreateSecretFull, isError: false, verifier: nil},
		{name: "ssh full", request: sshCreateSecretFull, isError: false, verifier: nil},

		{name: "aws missing key", request: awsMissingKey, isError: true, verifier: nil},
		{name: "aks missing key", request: aksMissingKey, isError: true, verifier: nil},
		{name: "gke missing key", request: gkeMissingKey, isError: true, verifier: nil},
		{name: "ssh missing key", request: sshMissingKey, isError: true, verifier: nil},
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
	AzureClientId       = "AZURE_CLIENT_ID"
	AzureClientSecret   = "AZURE_CLIENT_SECRET"
	AzureTenantId       = "AZURE_TENANT_ID"
	AzureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
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

var (
	awsCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Amazon,
		Values: map[string]string{
			pipConstants.AwsAccessKeyId:     AwsAccessKeyId,
			pipConstants.AwsSecretAccessKey: AwsSecretAccessKey,
		},
	}

	awsMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Amazon,
		Values: map[string]string{
			pipConstants.AwsSecretAccessKey: AwsSecretAccessKey,
		},
	}

	aksCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Azure,
		Values: map[string]string{
			pipConstants.AzureClientId:       AzureClientId,
			pipConstants.AzureClientSecret:   AzureClientSecret,
			pipConstants.AzureTenantId:       AzureTenantId,
			pipConstants.AzureSubscriptionId: AzureSubscriptionId,
		},
	}

	aksMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Azure,
		Values: map[string]string{
			pipConstants.AzureClientId:       AzureClientId,
			pipConstants.AzureSubscriptionId: AzureSubscriptionId,
		},
	}

	gkeCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Google,
		Values: map[string]string{
			pipConstants.Type:          gkeType,
			pipConstants.ProjectId:     gkeProjectId,
			pipConstants.PrivateKeyId:  gkePrivateKeyId,
			pipConstants.PrivateKey:    gkePrivateKey,
			pipConstants.ClientEmail:   gkeClientEmail,
			pipConstants.ClientId:      gkeClientId,
			pipConstants.AuthUri:       gkeAuthUri,
			pipConstants.TokenUri:      gkeTokenUri,
			pipConstants.AuthX509Url:   gkeAuthCert,
			pipConstants.ClientX509Url: gkeClientCert,
		},
	}

	gkeMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Google,
		Values: map[string]string{
			pipConstants.Type:          gkeType,
			pipConstants.ProjectId:     gkeProjectId,
			pipConstants.PrivateKeyId:  gkePrivateKeyId,
			pipConstants.PrivateKey:    gkePrivateKey,
			pipConstants.ClientId:      gkeClientId,
			pipConstants.AuthUri:       gkeAuthUri,
			pipConstants.TokenUri:      gkeTokenUri,
			pipConstants.AuthX509Url:   gkeAuthCert,
			pipConstants.ClientX509Url: gkeClientCert,
		},
	}

	sshCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pipConstants.SshSecretType,
		Values: map[string]string{
			pipConstants.User:                 SshUser,
			pipConstants.Identifier:           SshIdentifier,
			pipConstants.PublicKeyData:        SshPublicKeyData,
			pipConstants.PublicKeyFingerprint: SshPublicKeyFingerprint,
			pipConstants.PrivateKeyData:       SshPrivateKeyData,
		},
	}

	sshMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: pipConstants.SshSecretType,
		Values: map[string]string{
			pipConstants.User:                 SshUser,
			pipConstants.Identifier:           SshIdentifier,
			pipConstants.PublicKeyData:        SshPublicKeyData,
			pipConstants.PublicKeyFingerprint: SshPublicKeyFingerprint,
		},
	}
)

var (
	secretItem1 = secret.SecretsItemResponse{
		ID:   secretId,
		Name: secretDesc,
		Type: btypes.Google,
		Values: map[string]string{
			pipConstants.ProjectId: gkeProjectId,
		},
	}
)
