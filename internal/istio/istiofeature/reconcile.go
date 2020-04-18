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
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) Reconcile() error {
	desiredState := DesiredStatePresent
	if !m.Configuration.enabled {
		desiredState = DesiredStateAbsent
	}

	var reconcilers []ReconcilerWithCluster

	switch desiredState {
	case DesiredStatePresent:
		reconcilers = []ReconcilerWithCluster{
			m.ReconcileIstioOperatorNamespace,
			m.ReconcileIstioOperator,
			m.ReconcileIstio,
			m.ReconcileBackyardsNamespace,
			func(desiredState DesiredState, c cluster.CommonCluster) error {
				return m.ReconcileBackyards(desiredState, c, false)
			},
			m.ReconcileCanaryOperatorNamespace,
			m.ReconcileCanaryOperator,
			m.ReconcileNodeExporter,
			m.ReconcileRemoteIstios,
		}
	case DesiredStateAbsent:
		reconcilers = []ReconcilerWithCluster{
			m.ReconcileRemoteIstios,
			m.ReconcileIstio,
			m.ReconcileIstioOperator,
			m.ReconcileCanaryOperator,
			func(desiredState DesiredState, c cluster.CommonCluster) error {
				return m.ReconcileBackyards(desiredState, c, false)
			},
			m.ReconcileCanaryOperatorNamespace,
			m.ReconcileBackyardsNamespace,
			m.ReconcileIstioOperatorNamespace,
			m.ReconcileNodeExporter,
		}
	}

	for _, res := range reconcilers {
		err := res(desiredState, m.Master)
		if err != nil {
			return errors.WrapIf(err, "could not reconcile")
		}
	}

	return nil
}
