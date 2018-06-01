package secret_test

import (
	"reflect"
	"testing"

	btypes "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/secret"
)

func TestGetValue(t *testing.T) {

	cases := []struct {
		name          string
		secretItem    secret.SecretsItemResponse
		searchedKey   string
		expectedValue string
	}{
		{name: "gke project id", secretItem: secretItem1, searchedKey: secret.ProjectId, expectedValue: gkeProjectId},
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
		name    string
		request secret.CreateSecretRequest
		isError bool
	}{
		{name: "aws full", request: awsCreateSecretFull, isError: false},
		{name: "aks full", request: aksCreateSecretFull, isError: false},
		{name: "gke full", request: gkeCreateSecretFull, isError: false},
		{name: "ssh full", request: sshCreateSecretFull, isError: false},

		{name: "aws missing key", request: awsMissingKey, isError: true},
		{name: "aks missing key", request: aksMissingKey, isError: true},
		{name: "gke missing key", request: gkeMissingKey, isError: true},
		{name: "ssh missing key", request: sshMissingKey, isError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()

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
			secret.AwsAccessKeyId:     AwsAccessKeyId,
			secret.AwsSecretAccessKey: AwsSecretAccessKey,
		},
	}

	awsMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Amazon,
		Values: map[string]string{
			secret.AwsSecretAccessKey: AwsSecretAccessKey,
		},
	}

	aksCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Azure,
		Values: map[string]string{
			secret.AzureClientId:       AzureClientId,
			secret.AzureClientSecret:   AzureClientSecret,
			secret.AzureTenantId:       AzureTenantId,
			secret.AzureSubscriptionId: AzureSubscriptionId,
		},
	}

	aksMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Azure,
		Values: map[string]string{
			secret.AzureClientId:       AzureClientId,
			secret.AzureSubscriptionId: AzureSubscriptionId,
		},
	}

	gkeCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Google,
		Values: map[string]string{
			secret.Type:          gkeType,
			secret.ProjectId:     gkeProjectId,
			secret.PrivateKeyId:  gkePrivateKeyId,
			secret.PrivateKey:    gkePrivateKey,
			secret.ClientEmail:   gkeClientEmail,
			secret.ClientId:      gkeClientId,
			secret.AuthUri:       gkeAuthUri,
			secret.TokenUri:      gkeTokenUri,
			secret.AuthX509Url:   gkeAuthCert,
			secret.ClientX509Url: gkeClientCert,
		},
	}

	gkeMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: btypes.Google,
		Values: map[string]string{
			secret.Type:          gkeType,
			secret.ProjectId:     gkeProjectId,
			secret.PrivateKeyId:  gkePrivateKeyId,
			secret.PrivateKey:    gkePrivateKey,
			secret.ClientId:      gkeClientId,
			secret.AuthUri:       gkeAuthUri,
			secret.TokenUri:      gkeTokenUri,
			secret.AuthX509Url:   gkeAuthCert,
			secret.ClientX509Url: gkeClientCert,
		},
	}

	sshCreateSecretFull = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: secret.SshSecretType,
		Values: map[string]string{
			secret.User:                 SshUser,
			secret.Identifier:           SshIdentifier,
			secret.PublicKeyData:        SshPublicKeyData,
			secret.PublicKeyFingerprint: SshPublicKeyFingerprint,
			secret.PrivateKeyData:       SshPrivateKeyData,
		},
	}

	sshMissingKey = secret.CreateSecretRequest{
		Name: secretDesc,
		Type: secret.SshSecretType,
		Values: map[string]string{
			secret.User:                 SshUser,
			secret.Identifier:           SshIdentifier,
			secret.PublicKeyData:        SshPublicKeyData,
			secret.PublicKeyFingerprint: SshPublicKeyFingerprint,
		},
	}
)

var (
	secretItem1 = secret.SecretsItemResponse{
		ID:   secretId,
		Name: secretDesc,
		Type: btypes.Google,
		Values: map[string]string{
			secret.ProjectId: gkeProjectId,
		},
	}
)
