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

import "github.com/goph/emperror"

func (m *MeshReconciler) Reconcile() error {
	desiredState := DesiredStatePresent
	if !m.Configuration.enabled {
		desiredState = DesiredStateAbsent
	}

	var reconcilers []Reconciler

	switch desiredState {
	case DesiredStatePresent:
		reconcilers = []Reconciler{
			m.ReconcileNamespace,
			m.ReconcileMonitoring,
			m.ReconcilePrometheusScrapeConfig,
			m.ReconcileGrafanaDashboards,
			m.ReconcileIstioOperator,
			m.ReconcileIstio,
			m.ReconcileRemoteIstios,
			m.ReconcileUistioNamespace,
			m.ReconcileUistio,
		}
	case DesiredStateAbsent:
		reconcilers = []Reconciler{
			m.ReconcilePrometheusScrapeConfig,
			m.ReconcileGrafanaDashboards,
			m.ReconcileRemoteIstios,
			m.ReconcileIstio,
			m.ReconcileIstioOperator,
			m.ReconcileUistio,
			m.ReconcileUistioNamespace,
			m.ReconcileNamespace,
		}
	}

	for _, res := range reconcilers {
		err := res(desiredState)
		if err != nil {
			return emperror.Wrap(err, "could not reconcile")
		}
	}

	return nil
}
