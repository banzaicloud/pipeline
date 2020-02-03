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
// nolint: gochecknoglobals
var HookMap = map[string]PostFunctioner{
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
	pkgCluster.RestoreFromBackup: &PostFunctionWithParam{
		f:            RestoreFromBackup,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InitSpotConfig: &BasePostFunction{
		f:            InitSpotConfig,
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.DeployInstanceTerminationHandler: &BasePostFunction{
		f:            DeployInstanceTerminationHandler,
		ErrorHandler: ErrorHandler{},
	},
}

// BasePostHookFunctions default posthook functions after cluster create
// nolint: gochecknoglobals
var BasePostHookFunctions = []string{
	pkgCluster.InstallIngressControllerPostHook,
	pkgCluster.InstallKubernetesDashboardPostHook,
	pkgCluster.InstallClusterAutoscalerPostHook,
	pkgCluster.InstallHorizontalPodAutoscalerPostHook,
	pkgCluster.InitSpotConfig,
	pkgCluster.DeployInstanceTerminationHandler,
}

// PostFunctioner manages posthook functions
type PostFunctioner interface {
	Do(CommonCluster) error
	GetPriority() int
	Error(CommonCluster, error)
}

// ErrorHandler is the common struct which implement Error function
type ErrorHandler struct {
}

func (*ErrorHandler) Error(c CommonCluster, err error) {
	_ = c.SetStatus(pkgCluster.Error, err.Error())
}

// Priority can be used to run post hooks in a specific order
type Priority struct {
	priority int
}

// Priority returns the priority value of a posthook - the lower the value, the sooner the posthook will run
func (p *Priority) GetPriority() int {
	return p.priority
}

// BasePostFunction describe a default posthook function
type BasePostFunction struct {
	f      func(CommonCluster, pkgCluster.PostHookConfig) error
	config pkgCluster.PostHookConfig
	Priority
	ErrorHandler
}

// PostFunctionWithParam describes a posthook function with params
type PostFunctionWithParam struct {
	f      func(CommonCluster, pkgCluster.PostHookParam) error
	params pkgCluster.PostHookParam
	Priority
	ErrorHandler
}

// Do call function and pass CommonCluster and posthookParams
func (p *PostFunctionWithParam) Do(cluster CommonCluster) error {
	return p.f(cluster, p.params)
}

// Do call function and pass CommonCluster as param
func (b *BasePostFunction) Do(cluster CommonCluster) error {
	return b.f(cluster, b.config)
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
