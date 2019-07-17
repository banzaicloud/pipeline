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

	"emperror.dev/emperror"
	"github.com/spf13/viper"
	v1 "k8s.io/api/rbac/v1"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"

	pipConfig "github.com/banzaicloud/pipeline/config"
)

const federationClusterRoleBindingName = "feddns-rolebinding"
const externalDNSServiceAccount = "dns-external-dns"
const federationClusterRoleName = "kubefed-role"

func (m *FederationReconciler) ReconcileCRBForExtDNS(desiredState DesiredState) error {
	m.logger.Debug("start reconciling Federation controller")
	defer m.logger.Debug("finished reconciling Federation controller")

	if desiredState == DesiredStatePresent {
		err := m.createClusterRoleBindingForExternalDNS()
		if err != nil {
			return emperror.Wrap(err, "error creating ClusterRoleBinding for ExternalDNS")
		}
	} else {
		err := m.deleteClusterRoleBindingForExternalDNS()
		if err != nil {
			return emperror.Wrap(err, "error deleting ClusterRoleBinding for ExternalDNS")
		}
	}

	return nil
}

func (m *FederationReconciler) createClusterRoleBindingForExternalDNS() error {

	m.logger.Debug("start creating ClusterRoleBinding for ExternalDNS")
	defer m.logger.Debug("finished creating ClusterRoleBinding for ExternalDNS")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}
	cl, err := rbacv1.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	crb, err := cl.ClusterRoleBindings().Get(federationClusterRoleBindingName, apiv1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Warnf("ClusterRoleBinding for ExternalDNS not found, will try to create")
		} else {
			return err
		}
	} else if crb.Name == federationClusterRoleBindingName {
		m.logger.Debug("ClusterRoleBinding for ExternalDNS found")
		return nil
	}

	infraNamespace := viper.GetString(pipConfig.PipelineSystemNamespace)

	crb = &v1.ClusterRoleBinding{
		ObjectMeta: apiv1.ObjectMeta{
			Name: federationClusterRoleBindingName,
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     federationClusterRoleName,
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      externalDNSServiceAccount,
				Namespace: infraNamespace,
			},
		},
	}
	_, err = cl.ClusterRoleBindings().Create(crb)
	if err != nil {
		return err
	}

	return nil
}

func (m *FederationReconciler) deleteClusterRoleBindingForExternalDNS() error {

	m.logger.Debug("start deleting ClusterRoleBinding for ExternalDNS")
	defer m.logger.Debug("finished deleting ClusterRoleBinding for ExternalDNS")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}
	cl, err := rbacv1.NewForConfig(clientConfig)
	if err != nil {
		return err
	}
	err = cl.ClusterRoleBindings().Delete(federationClusterRoleBindingName, &apiv1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Warnf("crb for externalDND not found")
		} else {
			return err
		}
	}

	return nil
}
