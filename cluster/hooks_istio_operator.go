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

package cluster

import (
	pConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
)

const istioOperatorNamespace = "istio-system"
const istioOperatorDeploymentName = pkgHelm.BanzaiRepository + "/" + "istio-operator"
const istioOperatorReleaseName = "istio-operator"

// InstallIstioOperator is a posthook for installing istio-operator on a cluster
func InstallIstioOperator(cluster CommonCluster, param cluster.PostHookParam) error {
	err := installDeployment(
		cluster,
		istioOperatorNamespace,
		istioOperatorDeploymentName,
		istioOperatorReleaseName,
		[]byte{},
		viper.GetString(pConfig.IstioOperatorChartVersion),
		true)
	if err != nil {
		return emperror.Wrap(err, "installing istio-operator failed")
	}

	return nil
}
