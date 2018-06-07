package verify

import (
	"github.com/banzaicloud/banzai-types/constants"
	pipConstants "github.com/banzaicloud/pipeline/constants"
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
			cloudType: constants.Amazon,
			values:    awsCredentialsMap,
			verifier: &awsVerify{
				credentials: CreateAWSCredentials(awsCredentialsMap),
			},
		},
		{
			name:      "aks validator",
			cloudType: constants.Azure,
			values:    aksCredentialsMap,
			verifier: &aksVerify{
				credential: CreateAKSCredentials(aksCredentialsMap),
			},
		},
		{
			name:      "gke validator",
			cloudType: constants.Google,
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
		pipConstants.AwsAccessKeyId:     testAwsAccessKeyId,
		pipConstants.AwsSecretAccessKey: testAwsSecretAccessKey,
	}

	aksCredentialsMap = map[string]string{
		pipConstants.AzureClientId:       testAzureClientId,
		pipConstants.AzureClientSecret:   testAzureClientSecret,
		pipConstants.AzureTenantId:       testAzureTenantId,
		pipConstants.AzureSubscriptionId: testAzureSubscriptionId,
	}

	gkeCredentialsMap = map[string]string{
		pipConstants.Type:          testType,
		pipConstants.ProjectId:     testProjectId,
		pipConstants.PrivateKeyId:  testPrivateKeyId,
		pipConstants.PrivateKey:    testPrivateKey,
		pipConstants.ClientEmail:   testClientEmail,
		pipConstants.ClientId:      testClientId,
		pipConstants.AuthUri:       testAuthUri,
		pipConstants.TokenUri:      testTokenUri,
		pipConstants.AuthX509Url:   testAuthX509Url,
		pipConstants.ClientX509Url: testClientX509Url,
	}
)
