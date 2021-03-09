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
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// HookMap for api hook endpoints
// nolint: gochecknoglobals
var HookMap = map[string]PostFunctioner{
	pkgCluster.InstallKubernetesDashboardPostHook: &KubernetesDashboardPostHook{
		ErrorHandler: ErrorHandler{},
	},
	pkgCluster.InitSpotConfig: &InitSpotConfigPostHook{
		ErrorHandler: ErrorHandler{},
	},
}

// BasePostHookFunctions default posthook functions after cluster create
// nolint: gochecknoglobals
var BasePostHookFunctions = []string{
	pkgCluster.InstallKubernetesDashboardPostHook,
	pkgCluster.InitSpotConfig,
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
