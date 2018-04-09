package client

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/sirupsen/logrus"
	"net/http"
)

const BaseUrl = "https://management.azure.com"

type AKSClient struct {
	azureSdk *cluster.Sdk
	logger   *logrus.Logger
	clientId string
	secret   string
}

// GetAKSClient creates an *AKSClient instance with the passed credentials and default logger
func GetAKSClient(credentials *cluster.AKSCredential) (*AKSClient, error) {

	azureSdk, err := cluster.Authenticate(credentials)
	if err != nil {
		return nil, err
	}
	aksClient := &AKSClient{
		clientId: azureSdk.ServicePrincipal.ClientID,
		secret:   azureSdk.ServicePrincipal.ClientSecret,
		azureSdk: azureSdk,
		logger:   getDefaultLogger(),
	}
	if aksClient.clientId == "" {
		return nil, utils.NewErr("clientID is missing")
	}
	if aksClient.secret == "" {
		return nil, utils.NewErr("secret is missing")
	}
	return aksClient, nil
}

// With sets logger
func (a *AKSClient) With(i interface{}) {
	if a != nil {
		switch i.(type) {
		case logrus.Logger:
			logger := i.(logrus.Logger)
			a.logger = &logger
		case *logrus.Logger:
			a.logger = i.(*logrus.Logger)
		}
	}
}

// getDefaultLogger return the default logger
func getDefaultLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel
	logger.Formatter = new(logrus.JSONFormatter)
	return logger
}

func (a *AKSClient) List() ([]containerservice.ManagedCluster, error) {
	page, err := a.azureSdk.ManagedClusterClient.List(context.Background())
	if err != nil {
		return nil, err
	}
	return page.Values(), nil
}

func (a *AKSClient) CreateOrUpdate(request *cluster.CreateClusterRequest, managedCluster *containerservice.ManagedCluster) (*containerservice.ManagedCluster, error) {

	res, err := a.azureSdk.ManagedClusterClient.CreateOrUpdate(context.Background(), request.ResourceGroup, request.Name, *managedCluster)
	if err != nil {
		return nil, err
	}

	mc, err := res.Result(*a.azureSdk.ManagedClusterClient)
	if err != nil {
		return nil, err
	}

	return &mc, err
}

func (a *AKSClient) Delete(resourceGroup, name string) (*http.Response, error) {
	resp, err := a.azureSdk.ManagedClusterClient.Delete(context.Background(), resourceGroup, name)
	if err != nil {
		return nil, err
	}
	return resp.Response(), nil
}

func (a *AKSClient) Get(resourceGroup, name string) (containerservice.ManagedCluster, error) {
	return a.azureSdk.ManagedClusterClient.Get(context.Background(), resourceGroup, name)
}

func (a *AKSClient) GetAccessProfiles(resourceGroup, name, roleName string) (containerservice.ManagedClusterAccessProfile, error) {
	return a.azureSdk.ManagedClusterClient.GetAccessProfiles(context.Background(), resourceGroup, name, roleName)
}

func (a *AKSClient) ListVmSizes(location string) (result compute.VirtualMachineSizeListResult, err error) {
	return a.azureSdk.VMSizeClient.List(context.Background(), location)
}

func (a *AKSClient) ListLocations() (subscriptions.LocationListResult, error) {
	return a.azureSdk.SubscriptionsClient.ListLocations(context.Background(), a.azureSdk.ServicePrincipal.SubscriptionID)
}

func (a *AKSClient) ListVersions(location, resourceType string) (result containerservice.OrchestratorVersionProfileListResult, err error) {
	return a.azureSdk.ContainerServicesClient.ListOrchestrators(context.Background(), location, resourceType)
}

func (a *AKSClient) GetClientId() string {
	return a.clientId
}

func (a *AKSClient) GetClientSecret() string {
	return a.secret
}

func (a *AKSClient) LogDebug(args ...interface{}) {
	if a.logger != nil {
		a.logger.Debug(args...)
	}
}
func (a *AKSClient) LogInfo(args ...interface{}) {
	if a.logger != nil {
		a.logger.Info(args...)
	}
}
func (a *AKSClient) LogWarn(args ...interface{}) {
	if a.logger != nil {
		a.logger.Warn(args...)
	}
}
func (a *AKSClient) LogError(args ...interface{}) {
	if a.logger != nil {
		a.logger.Error(args...)
	}
}

func (a *AKSClient) LogFatal(args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatal(args...)
	}
}

func (a *AKSClient) LogPanic(args ...interface{}) {
	if a.logger != nil {
		a.logger.Panic(args...)
	}
}

func (a *AKSClient) LogDebugf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Debugf(format, args...)
	}
}

func (a *AKSClient) LogInfof(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Infof(format, args...)
	}
}

func (a *AKSClient) LogWarnf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Warnf(format, args...)
	}
}

func (a *AKSClient) LogErrorf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Errorf(format, args...)
	}
}

func (a *AKSClient) LogFatalf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatalf(format, args...)
	}
}

func (a *AKSClient) LogPanicf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Panicf(format, args...)
	}
}
