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

package api_test

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	"emperror.dev/emperror"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/secret/pkesecret"
	"github.com/banzaicloud/pipeline/internal/secret/restricted"
	"github.com/banzaicloud/pipeline/internal/secret/secretadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/types"
	clusterTypes "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		t.Skip("skipping as VAULT_ADDR is not explicitly defined")
	}

	vaultClient, err := vault.NewClient("pipeline")
	emperror.Panic(err)
	global.SetVault(vaultClient)

	secretStore := secretadapter.NewVaultStore(vaultClient, "secret")
	pkeSecreter := pkesecret.NewPkeSecreter(vaultClient, common.NoopLogger{})
	secretTypes := types.NewDefaultTypeList(types.DefaultTypeListConfig{
		TLSDefaultValidity: 365 * 24 * time.Hour,
		PkeSecreter:        pkeSecreter,
	})
	secret.InitSecretStore(secretStore, secretTypes)
	restricted.InitSecretStore(secret.Store)

	t.Run("testAddSecret", testAddSecret)
	t.Run("testDeleteSecrets", testDeleteSecrets)
	t.Run("testListSecrets", testListSecrets)
}

func testAddSecret(t *testing.T) {
	cases := []struct {
		name    string
		request secret.CreateSecretRequest
		isError bool
	}{
		{name: "add aws secret", request: awsCreateSecretRequest, isError: false},
		{name: "add aks secret", request: aksCreateSecretRequest, isError: false},
		{name: "add gke secret", request: gkeCreateSecretRequest, isError: false},

		{name: "add aws secret (missing key(s))", request: awsMissingKey, isError: true},
		{name: "add aks secret (missing key(s))", request: aksMissingKey, isError: true},
		{name: "add gke secret (missing key(s))", request: gkeMissingKey, isError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := secret.Store.Store(orgId, &tc.request)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func testListSecrets(t *testing.T) {
	_, _ = secret.Store.Store(orgId, &awsCreateSecretRequest)
	_, _ = secret.Store.Store(orgId, &aksCreateSecretRequest)
	_, _ = secret.Store.Store(orgId, &gkeCreateSecretRequest)

	cases := []struct {
		name           string
		secretType     string
		tag            []string
		expectedValues []*secret.SecretItemResponse
	}{
		{name: "List aws secrets", secretType: clusterTypes.Amazon, tag: awsCreateSecretRequest.Tags, expectedValues: awsExpectedItems},
		{name: "List aks secrets", secretType: clusterTypes.Azure, tag: aksCreateSecretRequest.Tags, expectedValues: aksExpectedItems},
		{name: "List gke secrets", secretType: clusterTypes.Google, tag: gkeCreateSecretRequest.Tags, expectedValues: gkeExpectedItems},
		{name: "List all secrets", secretType: "", tag: nil, expectedValues: allExpectedItems},
		{name: "List repo:pipeline secrets", secretType: "", tag: []string{"repo:pipeline"}, expectedValues: awsExpectedItems},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if items, err := secret.Store.List(orgId, &secret.ListSecretsQuery{Type: tc.secretType, Tags: tc.tag}); err != nil {
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
		})
	}
}

func testDeleteSecrets(t *testing.T) {
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

// nolint: gochecknoglobals
var (
	secretNameAmazon = "my-aws-secret"
	secretNameAzure  = "my-aks-secret"
	secretNameGoogle = "my-gke-secret"
)

// nolint: gochecknoglobals
var (
	secretIdAmazon = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameAmazon)))
	secretIdAzure  = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameAzure)))
	secretIdGoogle = fmt.Sprintf("%x", sha256.Sum256([]byte(secretNameGoogle)))
)

const (
	orgId = 12345
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
// nolint: gochecknoglobals
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
)

func toHiddenValues(secretType string) map[string]string {
	values := map[string]string{}
	for _, field := range secrettype.DefaultRules[secretType].Fields {
		values[field.Name] = "<hidden>"
	}
	return values
}

// Expected values after list
// nolint: gochecknoglobals
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
			Tags:    []string{},
			Version: 1,
		},
	}

	gkeExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdGoogle,
			Name:    secretNameGoogle,
			Type:    clusterTypes.Google,
			Values:  toHiddenValues(clusterTypes.Google),
			Tags:    []string{},
			Version: 1,
		},
	}

	allExpectedItems = []*secret.SecretItemResponse{
		{
			ID:      secretIdGoogle,
			Name:    secretNameGoogle,
			Type:    clusterTypes.Google,
			Values:  toHiddenValues(clusterTypes.Google),
			Tags:    []string{},
			Version: 1,
		},
		{
			ID:      secretIdAzure,
			Name:    secretNameAzure,
			Type:    clusterTypes.Azure,
			Values:  toHiddenValues(clusterTypes.Azure),
			Tags:    []string{},
			Version: 1,
		},
		{
			ID:      secretIdAmazon,
			Name:    secretNameAmazon,
			Type:    clusterTypes.Amazon,
			Values:  toHiddenValues(clusterTypes.Amazon),
			Tags:    awsCreateSecretRequest.Tags,
			Version: 1,
		},
	}
)
