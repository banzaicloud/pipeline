// Copyright © 2019 Banzai Cloud
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

package monitoring

import (
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/config"
)

type Configuration struct {
	pipelineSystemNamespace string
	grafanaAdminUsername    string
	headNodepoolName        string
	operator                struct {
		chartName    string
		chartVersion string
	}
	pushgateway struct {
		chartName    string
		chartVersion string
	}
}

func NewFeatureConfiguration() Configuration {
	return Configuration{
		pipelineSystemNamespace: viper.GetString(config.PipelineSystemNamespace),
		grafanaAdminUsername:    viper.GetString(config.MonitorGrafanaAdminUserNameKey),
		headNodepoolName:        viper.GetString(config.PipelineHeadNodePoolName),
		operator: struct {
			chartName    string
			chartVersion string
		}{
			chartName:    viper.GetString(config.PrometheusOperatorChartKey),
			chartVersion: viper.GetString(config.PrometheusOperatorVersionKey),
		},
		pushgateway: struct {
			chartName    string
			chartVersion string
		}{
			chartName:    viper.GetString(config.PrometheusPushgatewayChartKey),
			chartVersion: viper.GetString(config.PrometheusPushgatewayVersionKey),
		},
	}
}
