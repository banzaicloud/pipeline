package api_test

import (
	"reflect"
	"testing"

	btypes "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/secret"
)

func TestIsValidSecretType(t *testing.T) {

	cases := []struct {
		name       string
		secretType string
		error      error
	}{
		{name: "Amazon secret type", secretType: btypes.Amazon, error: nil},
		{name: "Azure secret type", secretType: btypes.Azure, error: nil},
		{name: "Google secret type", secretType: btypes.Google, error: nil},
		{name: "not supported secret type", secretType: invalidSecretType, error: api.ErrNotSupportedSecretType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := api.IsValidSecretType(tc.secretType)
			if !reflect.DeepEqual(tc.error, err) {
				t.Errorf("Expected error: %s, but got: %s", tc.error.Error(), err.Error())
			}
		})
	}

}

func TestListAllowedSecretTypes(t *testing.T) {

	cases := []struct {
		name             string
		secretType       string
		expectedResponse interface{}
		error
	}{
		{name: "List all allowed secret types", secretType: "", expectedResponse: allAllowedTypes, error: nil},
		{name: "List aws required keys", secretType: btypes.Amazon, expectedResponse: awsRequiredKeys, error: nil},
		{name: "List aks required keys", secretType: btypes.Azure, expectedResponse: aksRequiredKeys, error: nil},
		{name: "List gke required keys", secretType: btypes.Google, expectedResponse: gkeRequiredKeys, error: nil},
		{name: "Invalid secret type", secretType: invalidSecretType, expectedResponse: nil, error: api.ErrNotSupportedSecretType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if response, err := api.GetAllowedTypes(tc.secretType); err != nil {
				if !reflect.DeepEqual(tc.error, err) {
					t.Errorf("Error during listing allowed types: %s", err.Error())
				}
			} else {
				if !reflect.DeepEqual(tc.expectedResponse, response) {
					t.Errorf("Expected response: %s, but got: %s", tc.expectedResponse, response)
				}
			}
		})
	}

}

func TestAddSecret(t *testing.T) {

	cases := []struct {
		name     string
		request  secret.CreateSecretRequest
		secretId string
		isError  bool
	}{
		{name: "add aws secret", request: awsCreateSecretRequest, secretId: secretIdAmazon, isError: false},
		{name: "add aks secret", request: aksCreateSecretRequest, secretId: secretIdAzure, isError: false},
		{name: "add gke secret", request: gkeCreateSecretRequest, secretId: secretIdGoogle, isError: false},

		{name: "add aws secret (missing key(s))", request: awsMissingKey, isError: true},
		{name: "add aks secret (missing key(s))", request: aksMissingKey, isError: true},
		{name: "add gke secret (missing key(s))", request: gkeMissingKey, isError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.request.Validate(); err != nil {
				if !tc.isError {
					t.Errorf("Error during validate request: %s", err.Error())
				}
			} else {
				// valid request
				if err := secret.Store.Store(orgId, tc.secretId, tc.request); err != nil {
					t.Errorf("Error during save secret: %s", err.Error())
				}
			}
		})
	}

}

func TestListSecrets(t *testing.T) {

	cases := []struct {
		name           string
		secretType     string
		repoName       string
		expectedValues []secret.SecretsItemResponse
	}{
		{name: "List aws secrets", secretType: btypes.Amazon, repoName: "", expectedValues: awsExpectedItems},
		{name: "List aks secrets", secretType: btypes.Azure, repoName: "", expectedValues: aksExpectedItems},
		{name: "List gke secrets", secretType: btypes.Google, repoName: "", expectedValues: gkeExpectedItems},
		{name: "List all secrets", secretType: "", repoName: "", expectedValues: allExpectedItems},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := api.IsValidSecretType(tc.secretType); err != nil {
				t.Errorf("Error during validate secret type: %s", err)
			} else {
				if items, err := secret.Store.List(orgId, tc.secretType, tc.repoName, false); err != nil {
					t.Errorf("Error during listing secrets")
				} else {
					if !reflect.DeepEqual(tc.expectedValues, items) {
						t.Errorf("Expected values: %v, but got: %v", tc.expectedValues, items)
					}
				}
			}
		})
	}

}

func TestDeleteSecrets(t *testing.T) {

	cases := []struct {
		name     string
		secretId string
	}{
		{name: "Delete amazon secret", secretId: secretIdAmazon},
		{name: "Delete azure secret", secretId: secretIdAzure},
		{name: "Delete google secret", secretId: secretIdGoogle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := secret.Store.Delete(orgId, tc.secretId); err != nil {
				t.Errorf("Error during delete secret: %s", err.Error())
			}
		})
	}

}

const (
	orgId          = "12345"
	secretIdAmazon = "123"
	secretIdAzure  = "124"
	secretIdGoogle = "125"
)

const (
	secretName        = "testName1"
	invalidSecretType = "SOMETHING_ELSE"
)

// AWS test constants
const (
	awsAccessKeyId     = "testAccessKeyId"
	awsSecretAccessKey = "testSecretAccessKey"
)

// AKS test constants
const (
	aksClientId       = "testClientId"
	aksClientSecret   = "testClientSecret"
	aksTenantId       = "testTenantId"
	aksSubscriptionId = "testSubscriptionId"
)

// GKE test constants
const (
	gkeType         = "testType"
	gkeProjectId    = "testProject"
	gkePrivateKey   = "testPrivateKey"
	gkePrivateKeyId = "testPrivateKeyId"
	gkeEmail        = "testEmail"
	gkeId           = "testClientId"
	gkeAuthUri      = "testAuthUri"
	gkeTokenUri     = "testTokenUri"
	gkeAuthProvider = "testAuthProvider"
	gkeClientX509   = "testClientX509"
)

// Create requests
var (
	awsCreateSecretRequest = secret.CreateSecretRequest{
		Name:       secretName,
		SecretType: btypes.Amazon,
		Values: map[string]string{
			"AWS_ACCESS_KEY_ID":     awsAccessKeyId,
			"AWS_SECRET_ACCESS_KEY": awsSecretAccessKey,
		},
	}

	aksCreateSecretRequest = secret.CreateSecretRequest{
		Name:       secretName,
		SecretType: btypes.Azure,
		Values: map[string]string{
			"AZURE_CLIENT_ID":       aksClientId,
			"AZURE_CLIENT_SECRET":   aksClientSecret,
			"AZURE_TENANT_ID":       aksTenantId,
			"AZURE_SUBSCRIPTION_ID": aksSubscriptionId,
		},
	}

	gkeCreateSecretRequest = secret.CreateSecretRequest{
		Name:       secretName,
		SecretType: btypes.Google,
		Values: map[string]string{
			"type":                        gkeType,
			"project_id":                  gkeProjectId,
			"private_key_id":              gkePrivateKeyId,
			"private_key":                 gkePrivateKey,
			"client_email":                gkeEmail,
			"client_id":                   gkeId,
			"auth_uri":                    gkeAuthUri,
			"token_uri":                   gkeTokenUri,
			"auth_provider_x509_cert_url": gkeAuthProvider,
			"client_x509_cert_url":        gkeClientX509,
		},
	}

	awsMissingKey = secret.CreateSecretRequest{
		Name:       secretName,
		SecretType: btypes.Amazon,
		Values: map[string]string{
			"AWS_SECRET_ACCESS_KEY": awsSecretAccessKey,
		},
	}

	aksMissingKey = secret.CreateSecretRequest{
		Name:       secretName,
		SecretType: btypes.Azure,
		Values: map[string]string{
			"AZURE_CLIENT_ID":       aksClientId,
			"AZURE_TENANT_ID":       aksTenantId,
			"AZURE_SUBSCRIPTION_ID": aksSubscriptionId,
		},
	}

	gkeMissingKey = secret.CreateSecretRequest{
		Name:       secretName,
		SecretType: btypes.Google,
		Values: map[string]string{
			"type":                        gkeType,
			"project_id":                  gkeProjectId,
			"private_key_id":              gkePrivateKeyId,
			"auth_provider_x509_cert_url": gkeAuthProvider,
			"client_x509_cert_url":        gkeClientX509,
		},
	}
)

// Expected values after list
var (
	awsExpectedItems = []secret.SecretsItemResponse{
		{
			ID:         secretIdAmazon,
			Name:       secretName,
			SecretType: btypes.Amazon,
			Values:     nil,
		},
	}

	aksExpectedItems = []secret.SecretsItemResponse{
		{
			ID:         secretIdAzure,
			Name:       secretName,
			SecretType: btypes.Azure,
			Values:     nil,
		},
	}

	gkeExpectedItems = []secret.SecretsItemResponse{
		{
			ID:         secretIdGoogle,
			Name:       secretName,
			SecretType: btypes.Google,
			Values:     nil,
		},
	}

	allExpectedItems = []secret.SecretsItemResponse{
		{
			ID:         secretIdAmazon,
			Name:       secretName,
			SecretType: btypes.Amazon,
			Values:     nil,
		},
		{
			ID:         secretIdAzure,
			Name:       secretName,
			SecretType: btypes.Azure,
			Values:     nil,
		}, {
			ID:         secretIdGoogle,
			Name:       secretName,
			SecretType: btypes.Google,
			Values:     nil,
		},
	}
)

var (
	allAllowedTypes = secret.AllowedSecretTypesResponse{
		Allowed: secret.DefaultRules,
	}

	awsRequiredKeys = secret.AllowedFilteredSecretTypesResponse{
		Keys: secret.DefaultRules[btypes.Amazon],
	}

	aksRequiredKeys = secret.AllowedFilteredSecretTypesResponse{
		Keys: secret.DefaultRules[btypes.Azure],
	}

	gkeRequiredKeys = secret.AllowedFilteredSecretTypesResponse{
		Keys: secret.DefaultRules[btypes.Google],
	}
)
