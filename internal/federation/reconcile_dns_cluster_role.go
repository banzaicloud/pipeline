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

import (
	"strings"

	"emperror.dev/errors"
	v1 "k8s.io/api/rbac/v1"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

const kubefedDnsAPIGroup = "multiclusterdns.kubefed.k8s.io"

func (m *FederationReconciler) ReconcileClusterRoleForExtDNS(desiredState DesiredState) error {
	if desiredState == DesiredStatePresent {
		err := m.createClusterRoleForExternalDNS()
		if err != nil {
			return errors.Wrap(err, "error creating ClusterRole for ExternalDNS")
		}
	} else {
		err := m.deleteClusterRoleForExternalDNS()
		if err != nil {
			return errors.Wrap(err, "error deleting ClusterRole for ExternalDNS")
		}
	}
	return nil
}

func (m *FederationReconciler) createClusterRoleForExternalDNS() error {

	m.logger.Debug("start creating ClusterRole for ExternalDNS")
	defer m.logger.Debug("finished creating ClusterRole for ExternalDNS")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return errors.WithStackIf(err)
	}
	cl, err := rbacv1.NewForConfig(clientConfig)
	if err != nil {
		return errors.WithStackIf(err)
	}

	rb, err := cl.ClusterRoles().Get(federationDNSClusterRoleName, apiv1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Warnf("ClusterRole for ExternalDNS not found, will try to create")
		} else {
			return errors.WithStackIf(err)
		}
	} else if rb.Name == federationDNSClusterRoleName {
		m.logger.Debug("ClusterRole for ExternalDNS found")
		return nil
	}

	rb = &v1.ClusterRole{
		ObjectMeta: apiv1.ObjectMeta{
			Name: federationDNSClusterRoleName,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{kubefedDnsAPIGroup},
				Resources: []string{v1.ResourceAll},
				Verbs:     []string{"get", "watch", "list", "create", "update"},
			},
		},
	}
	_, err = cl.ClusterRoles().Create(rb)
	if err != nil {
		return errors.WithStackIf(err)
	}

	return nil
}

func (m *FederationReconciler) deleteClusterRoleForExternalDNS() error {

	m.logger.Debug("start deleting ClusterRole for ExternalDNS")
	defer m.logger.Debug("finished deleting ClusterRole for ExternalDNS")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return errors.WithStackIf(err)
	}
	cl, err := rbacv1.NewForConfig(clientConfig)
	if err != nil {
		return errors.WithStackIf(err)
	}
	err = cl.ClusterRoles().Delete(federationDNSClusterRoleName, &apiv1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Warnf("ClusterRole for externalDNS not found")
		} else {
			return errors.WithStackIf(err)
		}
	}

	return nil
}
