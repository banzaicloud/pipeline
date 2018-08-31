package cluster

import (
	"reflect"
	"runtime"
	"strings"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// HookMap for api hook endpoints
var HookMap = map[string]PostFunctioner{
	pkgCluster.StoreKubeConfig: &BasePostFunction{
		f:            StoreKubeConfig,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.SetupPrivileges: &BasePostFunction{
		f:            SetupPrivileges,
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
	pkgCluster.InstallKubernetesDashboardPostHook: &BasePostFunction{
		f:            InstallKubernetesDashboardPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallClusterAutoscalerPostHook: &BasePostFunction{
		f:            InstallClusterAutoscalerPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallHorizontalPodAutoscalerPostHook: &BasePostFunction{
		f:            InstallHorizontalPodAutoscalerPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallMonitoring: &BasePostFunction{
		f:            InstallMonitoring,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallLogging: &PostFunctionWithParam{
		f:            InstallLogging,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.RegisterDomainPostHook: &BasePostFunction{
		f:            RegisterDomainPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.LabelNodes: &BasePostFunction{
		f:            LabelNodes,
		ErrorHandler: ErrorHandler{},
	},
}

// BasePostHookFunctions default posthook functions after cluster create
var BasePostHookFunctions = []PostFunctioner{
	HookMap[pkgCluster.StoreKubeConfig],
	HookMap[pkgCluster.SetupPrivileges],
	HookMap[pkgCluster.UpdatePrometheusPostHook],
	HookMap[pkgCluster.InstallHelmPostHook],
	HookMap[pkgCluster.RegisterDomainPostHook],
	HookMap[pkgCluster.InstallIngressControllerPostHook],
	HookMap[pkgCluster.InstallKubernetesDashboardPostHook],
	HookMap[pkgCluster.InstallClusterAutoscalerPostHook],
	HookMap[pkgCluster.InstallHorizontalPodAutoscalerPostHook],
	HookMap[pkgCluster.LabelNodes],
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

// PostFunctionWithParam describes a posthook function with params
type PostFunctionWithParam struct {
	f      func(interface{}, pkgCluster.PostHookParam) error
	params pkgCluster.PostHookParam
	ErrorHandler
}

// Do call function and pass CommonCluster and posthookParams
func (p *PostFunctionWithParam) Do(cluster CommonCluster) error {
	return p.f(cluster, p.params)
}

// Do call function and pass CommonCluster as param
func (b *BasePostFunction) Do(cluster CommonCluster) error {
	return b.f(cluster)
}

func (b *BasePostFunction) String() string {
	return getFunctionName(b.f)
}

func getFunctionName(f interface{}) string {
	function := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	packageEnd := strings.LastIndex(function, ".")
	functionName := function[packageEnd+1:]

	return functionName
}

func (p *PostFunctionWithParam) String() string {
	return getFunctionName(p.f)
}

// SetParams sets posthook params
func (p *PostFunctionWithParam) SetParams(params pkgCluster.PostHookParam) {
	p.params = params
}
