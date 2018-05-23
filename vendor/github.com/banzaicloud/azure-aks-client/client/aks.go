package client

import (
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/sirupsen/logrus"
)

const BaseUrl = "https://management.azure.com"

type aksClient struct {
	azureSdk *cluster.Sdk
	logger   *logrus.Logger
	clientId string
	secret   string
}

// GetAKSClient creates an *aksClient instance with the passed credentials and default logger
func GetAKSClient(credentials *cluster.AKSCredential) (ClusterManager, error) {

	azureSdk, err := cluster.CreateSdk(credentials)
	if err != nil {
		return nil, err
	}
	aksClient := &aksClient{
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
func (a *aksClient) With(i interface{}) {
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

// getClientId returns client id
func (a *aksClient) getClientId() string {
	return a.clientId
}

// getClientSecret returns client secret
func (a *aksClient) getClientSecret() string {
	return a.secret
}

func (a *aksClient) LogDebug(args ...interface{}) {
	if a.logger != nil {
		a.logger.Debug(args...)
	}
}
func (a *aksClient) LogInfo(args ...interface{}) {
	if a.logger != nil {
		a.logger.Info(args...)
	}
}
func (a *aksClient) LogWarn(args ...interface{}) {
	if a.logger != nil {
		a.logger.Warn(args...)
	}
}
func (a *aksClient) LogError(args ...interface{}) {
	if a.logger != nil {
		a.logger.Error(args...)
	}
}

func (a *aksClient) LogFatal(args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatal(args...)
	}
}

func (a *aksClient) LogPanic(args ...interface{}) {
	if a.logger != nil {
		a.logger.Panic(args...)
	}
}

func (a *aksClient) LogDebugf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Debugf(format, args...)
	}
}

func (a *aksClient) LogInfof(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Infof(format, args...)
	}
}

func (a *aksClient) LogWarnf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Warnf(format, args...)
	}
}

func (a *aksClient) LogErrorf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Errorf(format, args...)
	}
}

func (a *aksClient) LogFatalf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatalf(format, args...)
	}
}

func (a *aksClient) LogPanicf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Panicf(format, args...)
	}
}
