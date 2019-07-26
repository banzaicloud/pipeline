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

package federation

import "emperror.dev/emperror"

func (m *FederationReconciler) Reconcile() error {
	desiredState := DesiredStatePresent
	if !m.Configuration.enabled {
		desiredState = DesiredStateAbsent
	}

	var reconcilers []Reconciler

	switch desiredState {
	case DesiredStatePresent:
		reconcilers = []Reconciler{
			m.ReconcileController,
			m.ReconcileMemberClusters,
		}
		// configure ext dns only if FederatedIngress or CrossClusterServiceDiscovery is enabled
		if m.Configuration.FederatedIngress || m.Configuration.CrossClusterServiceDiscovery {
			if !m.Configuration.GlobalScope {
				reconcilers = append(reconcilers, m.ReconcileClusterRoleForExtDNS)
			}
			reconcilers = append(reconcilers, m.ReconcileClusterRoleBindingForExtDNS)
			reconcilers = append(reconcilers, m.ReconcileExternalDNSController)
		}
	case DesiredStateAbsent:
		reconcilers = []Reconciler{
			m.ReconcileServiceDiscovery,
			m.ReconcileMemberClusters,
			m.ReconcileFederatedTypes,
			m.ReconcileController,
			m.ReconcileExternalDNSController,
		}
		if !m.Configuration.GlobalScope {
			reconcilers = append(reconcilers, m.ReconcileClusterRoleForExtDNS)
		}
		reconcilers = append(reconcilers, m.ReconcileClusterRoleBindingForExtDNS)
	}

	for _, res := range reconcilers {
		err := res(desiredState)
		if err != nil {
			return emperror.Wrap(err, "could not reconcile")
		}
	}

	return nil
}
