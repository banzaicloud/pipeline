// Copyright © 2020 Banzai Cloud
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
	"github.com/banzaicloud/pipeline/pkg/values"
)

type PostHookConfig struct {
	// ingress controller config
	Ingress IngressControllerConfig

	// Kubernetes Dashboard config
	Dashboard BasePostHookConfig

	// Init spot config
	Spotconfig SpotConfig

	// Instance Termination Handler config
	ITH BasePostHookConfig

	// Cluster Autoscaler config
	Autoscaler BaseConfig
}

type BasePostHookConfig struct {
	Enabled bool

	BaseChartConfig `mapstructure:",squash"`
}

type BaseChartConfig struct {
	Chart   string
	Version string
}

type SpotConfig struct {
	Enabled bool
	Charts  SpotChartsConfig
}

type SpotChartsConfig struct {
	Scheduler BaseChartConfig
	Webhook   BaseChartConfig
}

type BaseConfig struct {
	Enabled bool
}

type IngressControllerConfig struct {
	BasePostHookConfig `mapstructure:",squash"`

	Values values.Config
}
