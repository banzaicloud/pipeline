package api_test

import (
	"encoding/base64"
	"reflect"
	"testing"
	"time"

	btypes "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/api"
	pipConstants "github.com/banzaicloud/pipeline/constants"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
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
		isError  bool
		verifier verify.Verifier
	}{
		{name: "add aws secret", request: awsCreateSecretRequest, isError: false, verifier: nil},
		{name: "add aks secret", request: aksCreateSecretRequest, isError: false, verifier: nil},
		{name: "add gke secret", request: gkeCreateSecretRequest, isError: false, verifier: nil},

		{name: "add aws secret (missing key(s))", request: awsMissingKey, isError: true, verifier: nil},
		{name: "add aks secret (missing key(s))", request: aksMissingKey, isError: true, verifier: nil},
		{name: "add gke secret (missing key(s))", request: gkeMissingKey, isError: true, verifier: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.request.Validate(tc.verifier); err != nil {
				if !tc.isError {
					t.Errorf("Error during validate request: %s", err.Error())
				}
			} else {
				// valid request
				if _, err := secret.Store.Store(orgId, &tc.request); err != nil {
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
		tag            string
		expectedValues []*secret.SecretsItemResponse
	}{
		{name: "List aws secrets", secretType: btypes.Amazon, tag: "", expectedValues: awsExpectedItems},
		{name: "List aks secrets", secretType: btypes.Azure, tag: "", expectedValues: aksExpectedItems},
		{name: "List gke secrets", secretType: btypes.Google, tag: "", expectedValues: gkeExpectedItems},
		{name: "List all secrets", secretType: "", tag: "", expectedValues: allExpectedItems},
		{name: "List repo:pipeline secrets", secretType: "", tag: "repo:pipeline", expectedValues: awsExpectedItems},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := api.IsValidSecretType(tc.secretType); err != nil {
				t.Errorf("Error during validate secret type: %s", err)
			} else {
				if items, err := secret.Store.List(orgId, &secret.ListSecretsQuery{tc.secretType, tc.tag, false}); err != nil {
					t.Errorf("Error during listing secrets")
				} else {
					// Clear CreatedAt times, we don't know them
					for i := range items {
						items[i].CreatedAt = time.Time{}
					}
					if !reflect.DeepEqual(tc.expectedValues, items) {
						t.Errorf("\nExpected values: %v\nbut got: %v", tc.expectedValues, items)
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

var (
	secretNameAmazon = "my aws secret"
	secretNameAzure  = "my aks secret"
	secretNameGoogle = "my gke secret"
)

var (
	secretIdAmazon = base64.StdEncoding.EncodeToString([]byte(secretNameAmazon))
	secretIdAzure  = base64.StdEncoding.EncodeToString([]byte(secretNameAzure))
	secretIdGoogle = base64.StdEncoding.EncodeToString([]byte(secretNameGoogle))
)

const (
	orgId = "12345"
)

const (
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
		Name: secretNameAmazon,
		Type: btypes.Amazon,
		Values: map[string]string{
			"AWS_ACCESS_KEY_ID":     awsAccessKeyId,
			"AWS_SECRET_ACCESS_KEY": awsSecretAccessKey,
		},
		Tags: []string{"repo:pipeline"},
	}

	aksCreateSecretRequest = secret.CreateSecretRequest{
		Name: secretNameAzure,
		Type: btypes.Azure,
		Values: map[string]string{
			"AZURE_CLIENT_ID":       aksClientId,
			"AZURE_CLIENT_SECRET":   aksClientSecret,
			"AZURE_TENANT_ID":       aksTenantId,
			"AZURE_SUBSCRIPTION_ID": aksSubscriptionId,
		},
	}

	gkeCreateSecretRequest = secret.CreateSecretRequest{
		Name: secretNameGoogle,
		Type: btypes.Google,
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
		Name: secretNameAmazon,
		Type: btypes.Amazon,
		Values: map[string]string{
			"AWS_SECRET_ACCESS_KEY": awsSecretAccessKey,
		},
	}

	aksMissingKey = secret.CreateSecretRequest{
		Name: secretNameAzure,
		Type: btypes.Azure,
		Values: map[string]string{
			"AZURE_CLIENT_ID":       aksClientId,
			"AZURE_TENANT_ID":       aksTenantId,
			"AZURE_SUBSCRIPTION_ID": aksSubscriptionId,
		},
	}

	gkeMissingKey = secret.CreateSecretRequest{
		Name: secretNameGoogle,
		Type: btypes.Google,
		Values: map[string]string{
			"type":                        gkeType,
			"project_id":                  gkeProjectId,
			"private_key_id":              gkePrivateKeyId,
			"auth_provider_x509_cert_url": gkeAuthProvider,
			"client_x509_cert_url":        gkeClientX509,
		},
	}
)

func toHiddenValues(secretType string) map[string]string {
	values := map[string]string{}
	for _, key := range pipConstants.DefaultRules[secretType] {
		values[key] = "<hidden>"
	}
	return values
}

// Expected values after list
var (
	awsExpectedItems = []*secret.SecretsItemResponse{
		{
			ID:      secretIdAmazon,
			Name:    secretNameAmazon,
			Type:    btypes.Amazon,
			Values:  toHiddenValues(btypes.Amazon),
			Tags:    awsCreateSecretRequest.Tags,
			Version: 1,
		},
	}

	aksExpectedItems = []*secret.SecretsItemResponse{
		{
			ID:      secretIdAzure,
			Name:    secretNameAzure,
			Type:    btypes.Azure,
			Values:  toHiddenValues(btypes.Azure),
			Version: 1,
		},
	}

	gkeExpectedItems = []*secret.SecretsItemResponse{
		{
			ID:      secretIdGoogle,
			Name:    secretNameGoogle,
			Type:    btypes.Google,
			Values:  toHiddenValues(btypes.Google),
			Version: 1,
		},
	}

	allExpectedItems = []*secret.SecretsItemResponse{
		{
			ID:      secretIdGoogle,
			Name:    secretNameGoogle,
			Type:    btypes.Google,
			Values:  toHiddenValues(btypes.Google),
			Version: 1,
		},
		{
			ID:      secretIdAzure,
			Name:    secretNameAzure,
			Type:    btypes.Azure,
			Values:  toHiddenValues(btypes.Azure),
			Version: 1,
		}, {
			ID:      secretIdAmazon,
			Name:    secretNameAmazon,
			Type:    btypes.Amazon,
			Values:  toHiddenValues(btypes.Amazon),
			Tags:    awsCreateSecretRequest.Tags,
			Version: 1,
		},
	}
)

var (
	allAllowedTypes = secret.AllowedSecretTypesResponse{
		Allowed: pipConstants.DefaultRules,
	}

	awsRequiredKeys = secret.AllowedFilteredSecretTypesResponse{
		Keys: pipConstants.DefaultRules[btypes.Amazon],
	}

	aksRequiredKeys = secret.AllowedFilteredSecretTypesResponse{
		Keys: pipConstants.DefaultRules[btypes.Azure],
	}

	gkeRequiredKeys = secret.AllowedFilteredSecretTypesResponse{
		Keys: pipConstants.DefaultRules[btypes.Google],
	}
)
