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
	"github.com/goph/emperror"

	"github.com/banzaicloud/pipeline/internal/istio"
)

func (m *MeshReconciler) ReconcilePrometheusScrapeConfig(desiredState DesiredState) error {
	m.logger.Debug("reconciling Prometheus scrape configs")
	defer m.logger.Debug("Prometheus scrape configs reconciled")

	client, err := m.GetMasterK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		err := istio.AddPrometheusTargets(m.logger, client)
		if err != nil {
			return emperror.Wrap(err, "could not add prometheus targets")
		}
	} else {
		err := istio.RemovePrometheusTargets(m.logger, client)
		if err != nil {
			return emperror.Wrap(err, "could not remove prometheus targets")
		}
	}

	return nil
}
