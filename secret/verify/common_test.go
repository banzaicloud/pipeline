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
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
)

func TestNewVerifier(t *testing.T) {

	cases := []struct {
		name      string
		cloudType string
		values    map[string]string
		verifier  Verifier
	}{
		{
			name:      "aws validator",
			cloudType: pkgCluster.Amazon,
			values:    awsCredentialsMap,
			verifier: &awsVerify{
				credentials: CreateAWSCredentials(awsCredentialsMap),
			},
		},
		{
			name:      "aks validator",
			cloudType: pkgCluster.Azure,
			values:    aksCredentialsMap,
			verifier:  CreateAzureSecretVerifier(aksCredentialsMap),
		},
		{
			name:      "gke validator",
			cloudType: pkgCluster.Google,
			values:    gkeCredentialsMap,
			verifier:  CreateGCPSecretVerifier(gkeCredentialsMap),
		},
		{
			name:      "oci validator",
			cloudType: pkgCluster.Oracle,
			values:    OCICredentialMap,
			verifier:  oracle.CreateOCISecret(OCICredentialMap),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verifier := NewVerifier(tc.cloudType, tc.values)
			if !reflect.DeepEqual(verifier, tc.verifier) {
				t.Errorf("Expected verifier %v, but got: %v", tc.verifier, verifier)
				t.FailNow()
			}
		})
	}

}

const (
	testAwsAccessKeyId     = "testAwsAccessKeyId"
	testAwsSecretAccessKey = "testAwsSecretAccessKey"

	testAzureClientID       = "testAzureClientId"
	testAzureClientSecret   = "testAzureClientSecret"
	testAzureTenantID       = "testAzureTenantId"
	testAzureSubscriptionID = "testAzureSubscriptionId"

	testType          = "type"
	testProjectId     = "testProjectId"
	testPrivateKeyId  = "testPrivateKeyId"
	testPrivateKey    = "testPrivateKey"
	testClientEmail   = "testClientEmail"
	testClientId      = "testClientId"
	testAuthUri       = "testAuthUri"
	testTokenUri      = "testTokenUri"
	testAuthX509Url   = "testAuthX509Url"
	testClientX509Url = "testClientX509Url"

	testUserOCID           = "testUserOCID"
	testTenancyOCID        = "testTenancyOCID"
	testAPIKey             = "testAPIKey"
	testAPIKeyFringerprint = "testAPIKeyFringerprint"
	testRegion             = "testRegion"
)

// nolint: gochecknoglobals
var (
	awsCredentialsMap = map[string]string{
		secrettype.AwsAccessKeyId:     testAwsAccessKeyId,
		secrettype.AwsSecretAccessKey: testAwsSecretAccessKey,
	}

	aksCredentialsMap = map[string]string{
		secrettype.AzureClientID:       testAzureClientID,
		secrettype.AzureClientSecret:   testAzureClientSecret,
		secrettype.AzureTenantID:       testAzureTenantID,
		secrettype.AzureSubscriptionID: testAzureSubscriptionID,
	}

	gkeCredentialsMap = map[string]string{
		secrettype.Type:          testType,
		secrettype.ProjectId:     testProjectId,
		secrettype.PrivateKeyId:  testPrivateKeyId,
		secrettype.PrivateKey:    testPrivateKey,
		secrettype.ClientEmail:   testClientEmail,
		secrettype.ClientId:      testClientId,
		secrettype.AuthUri:       testAuthUri,
		secrettype.TokenUri:      testTokenUri,
		secrettype.AuthX509Url:   testAuthX509Url,
		secrettype.ClientX509Url: testClientX509Url,
	}

	OCICredentialMap = map[string]string{
		secrettype.OracleUserOCID:          testUserOCID,
		secrettype.OracleTenancyOCID:       testTenancyOCID,
		secrettype.OracleAPIKey:            testAPIKey,
		secrettype.OracleAPIKeyFingerprint: testAPIKeyFringerprint,
		secrettype.OracleRegion:            testRegion,
	}
)
