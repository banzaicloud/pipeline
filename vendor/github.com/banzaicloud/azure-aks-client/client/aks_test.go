package client

import (
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/sirupsen/logrus"
	"os"
	"reflect"
	"testing"
)

const (
	testClientId       = "AzureClientId"
	testClientSecret   = "AzureClientSecret"
	testSubscriptionId = "AzureSubscriptionId"
	testTenantId       = "AzureTenantId"
)

func TestDefaultLogger(t *testing.T) {
	logger := getDefaultLogger()

	expLogger := logrus.New()
	expLogger.Level = logrus.InfoLevel
	expLogger.Formatter = new(logrus.JSONFormatter)

	if !reflect.DeepEqual(&expLogger, &logger) {
		t.Errorf("Expexted logger %v, but got %v", expLogger, logger)
	}
}

func TestGetAKSClient(t *testing.T) {

	os.Setenv(cluster.AzureClientId, testClientId)
	os.Setenv(cluster.AzureClientSecret, testClientSecret)
	os.Setenv(cluster.AzureSubscriptionId, testSubscriptionId)
	os.Setenv(cluster.AzureTenantId, testTenantId)

	cases := []struct {
		name              string
		credentials       *cluster.AKSCredential
		expectedClient    *AKSClient
		withDefaultLogger bool
	}{
		{
			name:        "credentials from env",
			credentials: nil,
			expectedClient: &AKSClient{
				logger:   getDefaultLogger(),
				clientId: testClientId,
				secret:   testClientSecret,
			},
			withDefaultLogger: true,
		},
		{
			name: "auth with credentials directly",
			credentials: &cluster.AKSCredential{
				ClientId:       testClientId,
				ClientSecret:   testClientSecret,
				SubscriptionId: testSubscriptionId,
				TenantId:       testTenantId,
			},
			expectedClient: &AKSClient{
				logger:   getDefaultLogger(),
				clientId: testClientId,
				secret:   testClientSecret,
			},
			withDefaultLogger: true,
		},
		{
			name:        "with custom logger",
			credentials: nil,
			expectedClient: &AKSClient{
				logger:   getCustomLogger(),
				clientId: testClientId,
				secret:   testClientSecret,
			},
			withDefaultLogger: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			azureSdk, err := cluster.Authenticate(tc.credentials)
			if err != nil {
				t.Error("Error during authenticate")
				t.FailNow()
			}
			tc.expectedClient.azureSdk = azureSdk

			if cl, err := GetAKSClient(nil); err != nil {
				t.Errorf("Error during get aks client %v", err)
			} else {
				if !tc.withDefaultLogger {
					cl.With(getCustomLogger())
				}

				if !reflect.DeepEqual(tc.expectedClient, cl) {
					t.Errorf("Expected client %v, but got %v", tc.expectedClient, cl)
				}
			}
		})

	}
}

func getCustomLogger() *logrus.Logger {
	log := logrus.New()
	log.Level = logrus.ErrorLevel
	log.Formatter = new(logrus.TextFormatter)
	return log
}
