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
	"github.com/banzaicloud/pipeline/internal/global"
)

func (config Config) init() Config {
	config.internalConfig.canary = canaryOperatorConfiguration{
		chartVersion:    global.Config.Cluster.Backyards.Charts.CanaryOperator.Version,
		chartName:       global.Config.Cluster.Backyards.Charts.CanaryOperator.Chart,
		imageRepository: global.Config.Cluster.Backyards.Charts.CanaryOperator.Values.Operator.Image.Repository,
		imageTag:        global.Config.Cluster.Backyards.Charts.CanaryOperator.Values.Operator.Image.Tag,
	}

	config.internalConfig.backyards = backyardsConfiguration{
		chartVersion:    global.Config.Cluster.Backyards.Charts.Backyards.Version,
		chartName:       global.Config.Cluster.Backyards.Charts.Backyards.Chart,
		imageRepository: global.Config.Cluster.Backyards.Charts.Backyards.Values.Application.Image.Repository,
		imageTag:        global.Config.Cluster.Backyards.Charts.Backyards.Values.Application.Image.Tag,
		webImageTag:     global.Config.Cluster.Backyards.Charts.Backyards.Values.Web.Image.Tag,
	}

	config.internalConfig.istioOperator = istioOperatorConfiguration{
		chartVersion:    global.Config.Cluster.Backyards.Charts.IstioOperator.Version,
		chartName:       global.Config.Cluster.Backyards.Charts.IstioOperator.Chart,
		imageRepository: global.Config.Cluster.Backyards.Charts.IstioOperator.Values.Operator.Image.Repository,
		imageTag:        global.Config.Cluster.Backyards.Charts.IstioOperator.Values.Operator.Image.Tag,
		pilotImage:      global.Config.Cluster.Backyards.Istio.PilotImage,
		mixerImage:      global.Config.Cluster.Backyards.Istio.MixerImage,
	}

	return config
}
