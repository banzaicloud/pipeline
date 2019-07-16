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

package istiofeature

import (
	pConfig "github.com/banzaicloud/pipeline/config"
	"github.com/spf13/viper"
)

func (config Config) init() Config {
	config.internalConfig.iro = iroConfiguration{
		chartVersion:    viper.GetString(pConfig.IROChartVersion),
		chartName:       viper.GetString(pConfig.IROChartName),
		imageRepository: viper.GetString(pConfig.IROImageRepository),
		imageTag:        viper.GetString(pConfig.IROImageTag),
	}

	config.internalConfig.uistio = uistioConfiguration{
		chartVersion:    viper.GetString(pConfig.UistioChartVersion),
		chartName:       viper.GetString(pConfig.UistioChartName),
		imageRepository: viper.GetString(pConfig.UistioImageRepository),
		imageTag:        viper.GetString(pConfig.UistioImageTag),
	}

	config.internalConfig.istioOperator = istioOperatorConfiguration{
		chartVersion:    viper.GetString(pConfig.IstioOperatorChartVersion),
		chartName:       viper.GetString(pConfig.IstioOperatorChartName),
		imageRepository: viper.GetString(pConfig.IstioOperatorImageRepository),
		imageTag:        viper.GetString(pConfig.IstioOperatorImageTag),
		pilotImage:      viper.GetString(pConfig.IstioPilotImage),
		mixerImage:      viper.GetString(pConfig.IstioMixerImage),
	}

	return config
}
