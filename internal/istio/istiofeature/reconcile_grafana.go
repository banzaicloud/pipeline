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

func (m *MeshReconciler) ReconcileGrafanaDashboards(desiredState DesiredState) error {
	m.logger.Debug("reconciling Grafana dashboards")
	defer m.logger.Debug("Grafana dashboards reconciled")

	client, err := m.getMasterK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		err := istio.AddGrafanaDashboards(m.logger, client)
		if err != nil {
			return emperror.Wrap(err, "failed to add grafana dashboards")
		}
	} else {
		err := istio.DeleteGrafanaDashboards(m.logger, client)
		if err != nil {
			return emperror.Wrap(err, "failed to add grafana dashboards")
		}
	}

	return nil
}
