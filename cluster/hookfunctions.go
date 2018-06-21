package cluster

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"reflect"
	"runtime"
	"strings"
)

// HookMap for api hook endpoints
var HookMap = map[string]PostFunctioner{
	pkgCluster.StoreKubeConfig: &BasePostFunction{
		f:            StoreKubeConfig,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.PersistKubernetesKeys: &BasePostFunction{
		f:            PersistKubernetesKeys,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.UpdatePrometheusPostHook: &BasePostFunction{
		f:            UpdatePrometheusPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallHelmPostHook: &BasePostFunction{
		f:            InstallHelmPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallIngressControllerPostHook: &BasePostFunction{
		f:            InstallIngressControllerPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallClusterAutoscalerPostHook: &BasePostFunction{
		f:            InstallClusterAutoscalerPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallMonitoring: &BasePostFunction{
		f:            InstallMonitoring,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallLogging: &BasePostFunction{
		f:            InstallLogging,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.RegisterDomainPostHook: &BasePostFunction{
		f:            RegisterDomainPostHook,
		ErrorHandler: ErrorHandler{},
	},
}

// BasePostHookFunctions default posthook functions after cluster create
var BasePostHookFunctions = []PostFunctioner{
	HookMap[pkgCluster.StoreKubeConfig],
	HookMap[pkgCluster.PersistKubernetesKeys],
	HookMap[pkgCluster.UpdatePrometheusPostHook],
	HookMap[pkgCluster.InstallHelmPostHook],
	HookMap[pkgCluster.RegisterDomainPostHook],
	HookMap[pkgCluster.InstallIngressControllerPostHook],
	HookMap[pkgCluster.InstallClusterAutoscalerPostHook],
}

// RunPostHook describes a {cluster_id}/posthooks API request
type RunPostHook struct {
	Functions []string `json:"functions"`
}

// PostFunctioner manages posthook functions
type PostFunctioner interface {
	Do(CommonCluster) error
	Error(CommonCluster, error)
}

// ErrorHandler is the common struct which implement Error function
type ErrorHandler struct {
}

func (*ErrorHandler) Error(c CommonCluster, err error) {
	c.UpdateStatus(pkgCluster.Error, err.Error())
}

// BasePostFunction describe a default posthook function
type BasePostFunction struct {
	f func(interface{}) error
	ErrorHandler
}

// Do call function and pass CommonCluster as param
func (b *BasePostFunction) Do(cluster CommonCluster) error {
	return b.f(cluster)
}

func (b *BasePostFunction) String() string {

	function := runtime.FuncForPC(reflect.ValueOf(b.f).Pointer()).Name()
	packageEnd := strings.LastIndex(function, ".")
	functionName := function[packageEnd+1:]

	return functionName
}
