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
	"context"
	"encoding/base64"
	"strconv"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const (
	backoffDelaySeconds = 10
	backoffMaxretries   = 10
)

func (m *MeshReconciler) ReconcileRemoteIstios(desiredState DesiredState) error {
	m.logger.Debug("reconciling Remote Istios")
	defer m.logger.Debug("Remote Istios reconciled")

	remoteClusterIDs := make(map[uint]bool)
	if len(m.Remotes) > 0 {
		for _, remoteCluster := range m.Remotes {
			remoteClusterIDs[remoteCluster.GetID()] = true
			err := m.reconcileRemoteIstio(desiredState, remoteCluster)
			if err != nil {
				return err
			}
		}
	}

	clustersByRemoteIstios, err := m.getRemoteClustersByExistingRemoteIstioCRs()
	if err != nil {
		return err
	}

	for _, remoteCluster := range clustersByRemoteIstios {
		if remoteClusterIDs[remoteCluster.GetID()] == true {
			continue
		}

		err := m.reconcileRemoteIstio(DesiredStateAbsent, remoteCluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MeshReconciler) reconcileRemoteIstio(desiredState DesiredState, c cluster.CommonCluster) error {
	logger := m.logger.WithField("remoteClusterID", c.GetID())

	logger.Debug("reconciling Remote Istio")
	defer logger.Debug("Remote Istio reconciled")

	var reconcilers []ReconcilerWithCluster
	switch desiredState {
	case DesiredStatePresent:
		reconcilers = []ReconcilerWithCluster{
			m.reconcileRemoteIstioNamespace,
			m.reconcileRemoteIstioServiceAccount,
			m.reconcileRemoteIstioClusterRole,
			m.reconcileRemoteIstioClusterRoleBinding,
			m.reconcileRemoteIstioSecret,
			m.ReconcileRemoteIstio,
		}
	case DesiredStateAbsent:
		reconcilers = []ReconcilerWithCluster{
			m.ReconcileRemoteIstio,
			m.reconcileRemoteIstioClusterRoleBinding,
			m.reconcileRemoteIstioClusterRole,
			m.reconcileRemoteIstioServiceAccount,
			m.reconcileRemoteIstioNamespace,
			m.reconcileRemoteIstioSecret,
		}
	}

	for _, res := range reconcilers {
		err := res(desiredState, c)
		if err != nil {
			return emperror.Wrap(err, "could not reconcile")
		}
	}

	return nil
}

func (m *MeshReconciler) reconcileRemoteIstioSecret(desiredState DesiredState, c cluster.CommonCluster) error {
	secretName := c.GetName()

	client, err := m.GetK8sClient(m.Master)
	if err != nil {
		return err
	}

	if desiredState == DesiredStateAbsent {
		err := client.CoreV1().Secrets(istioOperatorNamespace).Delete(secretName, &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	kubeconfig, err := m.generateKubeconfig(c)
	if err != nil {
		return err
	}

	resource := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: istioOperatorNamespace,
		},
		Data: map[string][]byte{
			secretName: kubeconfig,
		},
	}

	_, err = client.CoreV1().Secrets(istioOperatorNamespace).Get(secretName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		return nil
	}
	_, err = client.CoreV1().Secrets(istioOperatorNamespace).Create(resource)
	if err != nil {
		return err
	}

	return nil
}

func (m *MeshReconciler) generateKubeconfig(c cluster.CommonCluster) ([]byte, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create rest config from kubeconfig")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "cloud not create client from kubeconfig")
	}

	sa, err := client.CoreV1().ServiceAccounts(istioOperatorNamespace).Get("istio-operator", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if len(sa.Secrets) == 0 {
		return nil, nil
	}

	secretName := sa.Secrets[0].Name

	secret, err := client.CoreV1().Secrets(istioOperatorNamespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	clusterName := c.GetName()

	yml := `apiVersion: v1
clusters:
   - cluster:
       certificate-authority-data: ` + base64.StdEncoding.EncodeToString(secret.Data["ca.crt"]) + `
       server: ` + config.Host + `
     name: ` + clusterName + `
contexts:
   - context:
       cluster: ` + clusterName + `
       user: ` + clusterName + `
     name: ` + clusterName + `
current-context: ` + clusterName + `
kind: Config
preferences: {}
users:
   - name: ` + clusterName + `
     user:
       token: ` + string(secret.Data["token"]) + `
`

	return []byte(yml), nil
}

func (m *MeshReconciler) reconcileRemoteIstioNamespace(desiredState DesiredState, c cluster.CommonCluster) error {
	client, err := m.GetK8sClient(c)
	if err != nil {
		return err
	}

	resource := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: istioOperatorNamespace,
		},
	}

	if desiredState == DesiredStatePresent {
		_, err := client.CoreV1().Namespaces().Get(istioOperatorNamespace, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			return nil
		}
		_, err = client.CoreV1().Namespaces().Create(resource)
		if err != nil {
			return err
		}
	} else {
		err := client.CoreV1().Namespaces().Delete(istioOperatorNamespace, &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *MeshReconciler) reconcileRemoteIstioServiceAccount(desiredState DesiredState, c cluster.CommonCluster) error {
	client, err := m.GetK8sClient(c)
	if err != nil {
		return err
	}

	resource := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
	}

	if desiredState == DesiredStatePresent {
		_, err := client.CoreV1().ServiceAccounts(istioOperatorNamespace).Get("istio-operator", metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			return nil
		}
		_, err = client.CoreV1().ServiceAccounts(istioOperatorNamespace).Create(resource)
		if err != nil {
			return err
		}
	} else {
		err := client.CoreV1().ServiceAccounts(istioOperatorNamespace).Delete("istio-operator", &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *MeshReconciler) reconcileRemoteIstioClusterRole(desiredState DesiredState, c cluster.CommonCluster) error {
	client, err := m.GetK8sClient(c)
	if err != nil {
		return err
	}

	resource := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				NonResourceURLs: []string{"*"},
				Verbs:           []string{"*"},
			},
		},
	}

	if desiredState == DesiredStatePresent {
		_, err := client.RbacV1().ClusterRoles().Get("istio-operator", metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			return nil
		}
		_, err = client.RbacV1().ClusterRoles().Create(resource)
		if err != nil {
			return err
		}
	} else {
		err := client.RbacV1().ClusterRoles().Delete("istio-operator", &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *MeshReconciler) reconcileRemoteIstioClusterRoleBinding(desiredState DesiredState, c cluster.CommonCluster) error {
	client, err := m.GetK8sClient(c)
	if err != nil {
		return err
	}

	resource := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "istio-operator",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "istio-operator",
				Namespace: istioOperatorNamespace,
			},
		},
	}

	if desiredState == DesiredStatePresent {
		_, err := client.RbacV1().ClusterRoleBindings().Get("istio-operator", metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			return nil
		}
		_, err = client.RbacV1().ClusterRoleBindings().Create(resource)
		if err != nil {
			return err
		}
	} else {
		err := client.RbacV1().ClusterRoleBindings().Delete("istio-operator", &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *MeshReconciler) getRemoteClustersByExistingRemoteIstioCRs() (map[uint]cluster.CommonCluster, error) {
	clusters := make(map[uint]cluster.CommonCluster, 0)

	client, err := m.GetMasterIstioOperatorK8sClient()
	if err != nil {
		return nil, err
	}

	remoteistios, err := client.IstioV1beta1().RemoteIstios(istioOperatorNamespace).List(metav1.ListOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, emperror.Wrap(err, "could not get remote istios")
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

		c, err := m.clusterGetter.GetClusterByID(context.Background(), m.Master.GetOrganizationId(), uint(clusterID))
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		clusters[c.GetID()] = c.(cluster.CommonCluster)
	}

	return clusters, nil
}
