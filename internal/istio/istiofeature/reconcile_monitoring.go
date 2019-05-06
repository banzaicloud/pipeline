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

	"github.com/banzaicloud/pipeline/cluster"
)

func (m *MeshReconciler) ReconcileMonitoring(desiredState DesiredState) error {
	m.logger.Debug("reconciling monitoring")
	defer m.logger.Debug("monitoring reconciled")

	if desiredState == DesiredStateAbsent {
		return nil
	}

	if m.Master.GetMonitoring() {
		m.logger.Debug("monitoring already installed")
	} else {
		m.logger.Debug("install monitoring")
		err := cluster.InstallMonitoring(m.Master)
		if err != nil {
			return emperror.Wrap(err, "could not install monitoring")
		}
		err = m.Master.Persist()
		if err != nil {
			return emperror.Wrap(err, "could not persist cluster state")
		}
	}

	return nil
}
