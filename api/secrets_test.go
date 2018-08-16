package api_test

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/banzaicloud/pipeline/api"
	clusterTypes "github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
)

func TestIsValidSecretType(t *testing.T) {

	cases := []struct {
		name       string
		secretType string
		error      error
	}{
		{name: "Amazon secret type", secretType: clusterTypes.Amazon, error: nil},
		{name: "Azure secret type", secretType: clusterTypes.Azure, error: nil},
		{name: "Google secret type", secretType: clusterTypes.Google, error: nil},
		{name: "Oracle secret type", secretType: clusterTypes.Oracle, error: nil},
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
		{name: "List aws required keys", secretType: clusterTypes.Amazon, expectedResponse: awsRequiredKeys, error: nil},
		{name: "List aks required keys", secretType: clusterTypes.Azure, expectedResponse: aksRequiredKeys, error: nil},
		{name: "List gke required keys", secretType: clusterTypes.Google, expectedResponse: gkeRequiredKeys, error: nil},
		{name: "List oci required keys", secretType: clusterTypes.Oracle, expectedResponse: OCIRequiredKeys, error: nil},
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
					t.Errorf("Expected response: %v, but got: %v", tc.expectedResponse, response)
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
		{name: "add oci secret", request: OCICreateSecretRequest, isError: false, verifier: nil},

		{name: "add aws secret (missing key(s))", request: awsMissingKey, isError: true, verifier: nil},
		{name: "add aks secret (missing key(s))", request: aksMissingKey, isError: true, verifier: nil},
		{name: "add gke secret (missing key(s))", request: gkeMissingKey, isError: true, verifier: nil},
		{name: "add oci secret (missing key(s))", request: OCIMissingKey, isError: true, verifier: nil},
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
		expectedValues []*secret.SecretItemResponse
	}{
		{name: "List aws secrets", secretType: clusterTypes.Amazon, tag: "", expectedValues: awsExpectedItems},
		{name: "List aks secrets", secretType: clusterTypes.Azure, tag: "", expectedValues: aksExpectedItems},
		{name: "List gke secrets", secretType: clusterTypes.Google, tag: "", expectedValues: gkeExpectedItems},
		{name: "List oci secrets", secretType: clusterTypes.Oracle, tag: "", expectedValues: OCIExpectedItems},
		{name: "List all secrets", secretType: "", tag: "", expectedValues: allExpectedItems},
		{name: "List repo:pipeline secrets", secretType: "", tag: "repo:pipeline", expectedValues: awsExpectedItems},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := api.IsValidSecretType(tc.secretType); err != nil {
				t.Errorf("Error during validate secret type: %s", err)
			} else {
				if items, err := secret.Store.List(orgId, &secretTypes.ListSecretsQuery{tc.secretType, tc.tag, false}); err != nil {
					t.Errorf("Error during listing secrets")
				} else {
					// Clear CreatedAt times, we don't know them
					for i := range items {
						items[i].UpdatedAt = time.Time{}
					}
					sort.Sort(sortableSecretItemResponses(tc.expectedValues))
					sort.Sort(sortableSecretItemResponses(items))
					if !reflect.DeepEqual(tc.expectedValues, items) {
						t.Errorf("\nExpected values: %#+v\nbut got: %#+v", tc.expectedValues, items)
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
		{name: "Delete oracle secret", secretId: secretIdOracle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := secret.Store.Delete(orgId, tc.secretId); err != nil {
				t.Errorf("Error during delete secret: %s", err.Error())
			}
		})
	}
}

type sortableSecretItemResponses []*secret.SecretItemResponse

func (s sortableSecretItemResponses) Len() int {
	return len(s)
}

func (s sortableSecretItemResponses) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortableSecretItemResponses) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

var (
	secretNameAmazon = "my-aws-secret"
	secretNameAzure  = "my-aks-secret"
	secretNameGoogle = "my-gke-secret"
	secretNameOracle = "my-oci-secret"
)

var (
	secretIdAmazon = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameAmazon)))
	secretIdAzure  = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameAzure)))
	secretIdGoogle = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameGoogle)))
	secretIdOracle = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameOracle)))
)

const (
	orgId = 12345
)

const (
	invalidSecretType = "SOMETHING_ELSE"
)

// AWS test constants
const (
	awsRegion          = "testRegion"
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

const (
	OCIUserOCID          = "OCIUserOCID"
	OCITenancyOCID       = "OCITenancyOCID"
	OCIAPIKey            = "OCIAPIKey"
	OCIAPIKeyFingerprint = "OCIAPIKeyFingerprint"
	OCIRegion            = "OCIRegion"
	OCICompartmentOCID   = "OCICompartmentOCID"
)

// Create requests
var (
	awsCreateSecretRequest = secret.CreateSecretRequest{
		Name: secretNameAmazon,
		Type: clusterTypes.Amazon,
		Values: map[string]string{
			"AWS_REGION":            awsRegion,
			"AWS_ACCESS_KEY_ID":     awsAccessKeyId,
			"AWS_SECRET_ACCESS_KEY": awsSecretAccessKey,
		},
		Tags: []string{"repo:pipeline"},
	}

	aksCreateSecretRequest = secret.CreateSecretRequest{
		Name: secretNameAzure,
		Type: clusterTypes.Azure,
		Values: map[string]string{
			"AZURE_CLIENT_ID":       aksClientId,
			"AZURE_CLIENT_SECRET":   aksClientSecret,
			"AZURE_TENANT_ID":       aksTenantId,
			"AZURE_SUBSCRIPTION_ID": aksSubscriptionId,
		},
	}

	gkeCreateSecretRequest = secret.CreateSecretRequest{
		Name: secretNameGoogle,
		Type: clusterTypes.Google,
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

	OCICreateSecretRequest = secret.CreateSecretRequest{
		Name: secretNameOracle,
		Type: clusterTypes.Oracle,
		Values: map[string]string{
			"user_ocid":           OCIUserOCID,
			"tenancy_ocid":        OCITenancyOCID,
			"api_key":             OCIAPIKey,
			"api_key_fingerprint": OCIAPIKeyFingerprint,
			"region":              OCIRegion,
			"compartment_ocid":    OCICompartmentOCID,
		},
	}

	awsMissingKey = secret.CreateSecretRequest{
		Name: secretNameAmazon,
		Type: clusterTypes.Amazon,
		Values: map[string]string{
			"AWS_SECRET_ACCESS_KEY": awsSecretAccessKey,
		},
	}

	aksMissingKey = secret.CreateSecretRequest{
		Name: secretNameAzure,
		Type: clusterTypes.Azure,
		Values: map[string]string{
			"AZURE_CLIENT_ID":       aksClientId,
			"AZURE_TENANT_ID":       aksTenantId,
			"AZURE_SUBSCRIPTION_ID": aksSubscriptionId,
		},
	}

	gkeMissingKey = secret.CreateSecretRequest{
		Name: secretNameGoogle,
		Type: clusterTypes.Google,
		Values: map[string]string{
			"type":                        gkeType,
			"project_id":                  gkeProjectId,
			"private_key_id":              gkePrivateKeyId,
			"auth_provider_x509_cert_url": gkeAuthProvider,
			"client_x509_cert_url":        gkeClientX509,
		},
	}

	OCIMissingKey = secret.CreateSecretRequest{
		Name: secretNameOracle,
		Type: clusterTypes.Oracle,
		Values: map[string]string{
			"tenancy_ocid":        OCITenancyOCID,
			"api_key":             OCIAPIKey,
			"api_key_fingerprint": OCIAPIKeyFingerprint,
			"region":              OCIRegion,
		},
	}
)

func toHiddenValues(secretType string) map[string]string {
	values := map[string]string{}
	for _, field := range secretTypes.DefaultRules[secretType].Fields {
		values[field.Name] = "<hidden>"
	}
	return values
}

// Expected values after list
var (
	awsExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdAmazon,
			Name:    secretNameAmazon,
			Type:    clusterTypes.Amazon,
			Values:  toHiddenValues(clusterTypes.Amazon),
			Tags:    awsCreateSecretRequest.Tags,
			Version: 1,
		},
	}

	aksExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdAzure,
			Name:    secretNameAzure,
			Type:    clusterTypes.Azure,
			Values:  toHiddenValues(clusterTypes.Azure),
			Version: 1,
		},
	}

	gkeExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdGoogle,
			Name:    secretNameGoogle,
			Type:    clusterTypes.Google,
			Values:  toHiddenValues(clusterTypes.Google),
			Version: 1,
		},
	}

	OCIExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdOracle,
			Name:    secretNameOracle,
			Type:    clusterTypes.Oracle,
			Values:  toHiddenValues(clusterTypes.Oracle),
			Version: 1,
		},
	}

	allExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdGoogle,
			Name:    secretNameGoogle,
			Type:    clusterTypes.Google,
			Values:  toHiddenValues(clusterTypes.Google),
			Version: 1,
		},
		{
			ID:      secretIdAzure,
			Name:    secretNameAzure,
			Type:    clusterTypes.Azure,
			Values:  toHiddenValues(clusterTypes.Azure),
			Version: 1,
		}, {
			ID:      secretIdAmazon,
			Name:    secretNameAmazon,
			Type:    clusterTypes.Amazon,
			Values:  toHiddenValues(clusterTypes.Amazon),
			Tags:    awsCreateSecretRequest.Tags,
			Version: 1,
		}, {
			ID:      secretIdOracle,
			Name:    secretNameOracle,
			Type:    clusterTypes.Oracle,
			Values:  toHiddenValues(clusterTypes.Oracle),
			Version: 1,
		},
	}
)

var (
	allAllowedTypes = secretTypes.DefaultRules

	awsRequiredKeys = secretTypes.DefaultRules[clusterTypes.Amazon]

	aksRequiredKeys = secretTypes.DefaultRules[clusterTypes.Azure]

	gkeRequiredKeys = secretTypes.DefaultRules[clusterTypes.Google]

	OCIRequiredKeys = secretTypes.DefaultRules[clusterTypes.Oracle]
)
