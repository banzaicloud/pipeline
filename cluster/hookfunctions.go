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
	pkgCluster.TaintHeadNodes: &BasePostFunction{
		f:            TaintHeadNodes,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallPVCOperator: &BasePostFunction{
		f:            InstallPVCOperatorPostHook,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InstallAnchoreImageValidator: &BasePostFunction{
		f:            InstallAnchoreImageValidator,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.RestoreFromBackup: &PostFunctionWithParam{
		f:            RestoreFromBackup,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InitSpotConfig: &BasePostFunction{
		f:            InitSpotConfig,
		ErrorHandler: ErrorHandler{},
	},
}

// BasePostHookFunctions default posthook functions after cluster create
var BasePostHookFunctions = []PostFunctioner{
	HookMap[pkgCluster.StoreKubeConfig],
	HookMap[pkgCluster.SetupPrivileges],
	HookMap[pkgCluster.LabelNodes],
	HookMap[pkgCluster.TaintHeadNodes],
	HookMap[pkgCluster.InstallHelmPostHook],
	HookMap[pkgCluster.RegisterDomainPostHook],
	HookMap[pkgCluster.InstallIngressControllerPostHook],
	HookMap[pkgCluster.InstallKubernetesDashboardPostHook],
	HookMap[pkgCluster.InstallClusterAutoscalerPostHook],
	HookMap[pkgCluster.InstallHorizontalPodAutoscalerPostHook],
	HookMap[pkgCluster.InstallPVCOperator],
	HookMap[pkgCluster.InstallAnchoreImageValidator],
	HookMap[pkgCluster.InitSpotConfig],
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
