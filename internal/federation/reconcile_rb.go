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

const federationRoleBindingName = "feddns-rb"
const externalDNSServiceAccount = "dns-external-dns"
const federationRoleName = "kubefed-role"

func (m *FederationReconciler) ReconcileRoleBindingForExtDNS(desiredState DesiredState) error {
	if desiredState == DesiredStatePresent {
		err := m.createRoleBindingForExternalDNS()
		if err != nil {
			return emperror.Wrap(err, "error creating RoleBinding for ExternalDNS")
		}
	} else {
		err := m.deleteRoleBindingForExternalDNS()
		if err != nil {
			return emperror.Wrap(err, "error deleting RoleBinding for ExternalDNS")
		}
	}

	return nil
}

func (m *FederationReconciler) createRoleBindingForExternalDNS() error {

	m.logger.Debug("start creating RoleBinding for ExternalDNS")
	defer m.logger.Debug("finished creating RoleBinding for ExternalDNS")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}
	cl, err := rbacv1.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	rb, err := cl.RoleBindings(m.Configuration.TargetNamespace).Get(federationRoleBindingName, apiv1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Warnf("RoleBinding for ExternalDNS not found, will try to create")
		} else {
			return err
		}
	} else if rb.Name == federationRoleBindingName {
		m.logger.Debug("RoleBinding for ExternalDNS found")
		return nil
	}

	infraNamespace := viper.GetString(pipConfig.PipelineSystemNamespace)

	rb = &v1.RoleBinding{
		ObjectMeta: apiv1.ObjectMeta{
			Name: federationRoleBindingName,
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     federationRoleName,
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      externalDNSServiceAccount,
				Namespace: infraNamespace,
			},
		},
	}
	_, err = cl.RoleBindings(m.Configuration.TargetNamespace).Create(rb)
	if err != nil {
		return err
	}

	return nil
}

func (m *FederationReconciler) deleteRoleBindingForExternalDNS() error {

	m.logger.Debug("start deleting RoleBinding for ExternalDNS")
	defer m.logger.Debug("finished deleting RoleBinding for ExternalDNS")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}
	cl, err := rbacv1.NewForConfig(clientConfig)
	if err != nil {
		return err
	}
	err = cl.RoleBindings(m.Configuration.TargetNamespace).Delete(federationRoleBindingName, &apiv1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Warnf("rb for externalDND not found")
		} else {
			return err
		}
	}

	return nil
}
