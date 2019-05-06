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
	"strconv"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *MeshReconciler) GetClusterStatus() (map[uint]string, error) {
	status := make(map[uint]string, 0)

	client, err := m.GetMasterIstioOperatorK8sClient()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get istio operator client")
	}

	istios, err := client.IstioV1beta1().Istios(istioOperatorNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, emperror.Wrap(err, "could not list istio CRs")
	}
	for _, istio := range istios.Items {
		labels := istio.GetLabels()
		if len(labels) == 0 {
			continue
		}

		cID := istio.Labels["cluster.banzaicloud.com/id"]
		if cID == "" {
			continue
		}

		clusterID, err := strconv.ParseUint(cID, 10, 64)
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		status[uint(clusterID)] = string(istio.Status.Status)
	}

	remoteistios, err := client.IstioV1beta1().RemoteIstios(istioOperatorNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, emperror.Wrap(err, "could not list istio CRs")
	}
	for _, remoteistio := range remoteistios.Items {
		labels := remoteistio.GetLabels()
		if len(labels) == 0 {
			continue
		}

		cID := remoteistio.Labels["cluster.banzaicloud.com/id"]
		if cID == "" {
			continue
		}

		clusterID, err := strconv.ParseUint(cID, 10, 64)
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		status[uint(clusterID)] = string(remoteistio.Status.Status)
	}

	return status, nil
}
