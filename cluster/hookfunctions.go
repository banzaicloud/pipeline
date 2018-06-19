package cluster

import (
	"github.com/banzaicloud/banzai-types/constants"
	pipConstants "github.com/banzaicloud/pipeline/constants"
	"reflect"
	"runtime"
	"strings"
)

// HookMap for api hook endpoints
var HookMap = map[string]PostFunctioner{
	pipConstants.StoreKubeConfig: &BasePostFunction{
		f:            StoreKubeConfig,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.PersistKubernetesKeys: &BasePostFunction{
		f:            PersistKubernetesKeys,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.UpdatePrometheusPostHook: &BasePostFunction{
		f:            UpdatePrometheusPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.InstallHelmPostHook: &BasePostFunction{
		f:            InstallHelmPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.InstallIngressControllerPostHook: &BasePostFunction{
		f:            InstallIngressControllerPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.InstallClusterAutoscalerPostHook: &BasePostFunction{
		f:            InstallClusterAutoscalerPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.InstallMonitoring: &BasePostFunction{
		f:            InstallMonitoring,
		ErrorHandler: ErrorHandler{},
	},
	pipConstants.InstallLogging: &BasePostFunction{
		f:            InstallLogging,
		ErrorHandler: ErrorHandler{},
	},
}

var BasePostHookFunctions = []PostFunctioner{
	HookMap[pipConstants.StoreKubeConfig],
	HookMap[pipConstants.PersistKubernetesKeys],
	HookMap[pipConstants.UpdatePrometheusPostHook],
	HookMap[pipConstants.InstallHelmPostHook],
	HookMap[pipConstants.InstallIngressControllerPostHook],
	HookMap[pipConstants.InstallClusterAutoscalerPostHook],
}

type PostFunctioner interface {
	Do(CommonCluster) error
	Error(CommonCluster, error)
}

type ErrorHandler struct {
}

func (*ErrorHandler) Error(c CommonCluster, err error) {
	c.UpdateStatus(constants.Error, err.Error())
}

type BasePostFunction struct {
	f func(interface{}) error
	ErrorHandler
}

func (b *BasePostFunction) Do(cluster CommonCluster) error {
	return b.f(cluster)
}

func (b *BasePostFunction) String() string {

	function := runtime.FuncForPC(reflect.ValueOf(b.f).Pointer()).Name()
	packageEnd := strings.LastIndex(function, ".")
	functionName := function[packageEnd+1:]

	return functionName
}
