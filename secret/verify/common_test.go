package verify

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"reflect"
	"testing"
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
			verifier: &aksVerify{
				credential: CreateAKSCredentials(aksCredentialsMap),
			},
		},
		{
			name:      "gke validator",
			cloudType: pkgCluster.Google,
			values:    gkeCredentialsMap,
			verifier: &gkeVerify{
				svc: CreateServiceAccount(gkeCredentialsMap),
			},
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

	testAzureClientId       = "testAzureClientId"
	testAzureClientSecret   = "testAzureClientSecret"
	testAzureTenantId       = "testAzureTenantId"
	testAzureSubscriptionId = "testAzureSubscriptionId"

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
)

var (
	awsCredentialsMap = map[string]string{
		pkgSecret.AwsAccessKeyId:     testAwsAccessKeyId,
		pkgSecret.AwsSecretAccessKey: testAwsSecretAccessKey,
	}

	aksCredentialsMap = map[string]string{
		pkgSecret.AzureClientId:       testAzureClientId,
		pkgSecret.AzureClientSecret:   testAzureClientSecret,
		pkgSecret.AzureTenantId:       testAzureTenantId,
		pkgSecret.AzureSubscriptionId: testAzureSubscriptionId,
	}

	gkeCredentialsMap = map[string]string{
		pkgSecret.Type:          testType,
		pkgSecret.ProjectId:     testProjectId,
		pkgSecret.PrivateKeyId:  testPrivateKeyId,
		pkgSecret.PrivateKey:    testPrivateKey,
		pkgSecret.ClientEmail:   testClientEmail,
		pkgSecret.ClientId:      testClientId,
		pkgSecret.AuthUri:       testAuthUri,
		pkgSecret.TokenUri:      testTokenUri,
		pkgSecret.AuthX509Url:   testAuthX509Url,
		pkgSecret.ClientX509Url: testClientX509Url,
	}
)
