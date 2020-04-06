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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"emperror.dev/errors"
	"github.com/banzaicloud/istio-operator/pkg/apis/istio/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) ReconcileRemoteIstios(desiredState DesiredState, c cluster.CommonCluster) error {
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

	clustersByRemoteIstios, err := m.getRemoteClustersByExistingRemoteIstioCRs(c)
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
			m.ReconcileBackyardsNamespace,
			m.reconcileRemoteIstioALSService,
			m.reconcileRemoteIstioTracingService,
			func(desiredState DesiredState, c cluster.CommonCluster) error {
				return m.ReconcileBackyards(desiredState, c, true)
			},
			m.ReconcileNodeExporter,
			m.reconcileRemoteIstioPrometheusService,
		}
	case DesiredStateAbsent:
		reconcilers = []ReconcilerWithCluster{
			m.ReconcileRemoteIstio,
			m.reconcileRemoteIstioClusterRoleBinding,
			m.reconcileRemoteIstioClusterRole,
			m.reconcileRemoteIstioServiceAccount,
			m.reconcileRemoteIstioNamespace,
			m.reconcileRemoteIstioSecret,
			func(desiredState DesiredState, c cluster.CommonCluster) error {
				return m.ReconcileBackyards(desiredState, c, true)
			},
			m.reconcileRemoteIstioALSService,
			m.reconcileRemoteIstioTracingService,
			m.ReconcileBackyardsNamespace,
			m.ReconcileNodeExporter,
			m.reconcileRemoteIstioPrometheusService,
		}
	}

	for _, res := range reconcilers {
		err := res(desiredState, c)
		if err != nil {
			return errors.WrapIf(err, "could not reconcile")
		}
	}

	return nil
}

func (m *MeshReconciler) reconcileRemoteIstioSecret(desiredState DesiredState, c cluster.CommonCluster) error {
	secretName := c.GetName()

	resource := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: istioOperatorNamespace,
		},
		Data: make(map[string][]byte),
	}

	client, err := m.getRuntimeK8sClient(m.Master)
	if err != nil {
		return err
	}

	if desiredState == DesiredStateAbsent {
		return errors.WithStack(m.deleteResource(client, resource))
	}

	kubeconfig, err := m.generateKubeconfig(c)
	if err != nil {
		return err
	}

	resource.Data[secretName] = kubeconfig

	return errors.WithStack(m.applyResource(client, resource))
}

func (m *MeshReconciler) generateKubeconfig(c cluster.CommonCluster) ([]byte, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get k8s config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "cloud not create client from kubeconfig")
	}

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return nil, errors.WrapIf(err, "cloud not create client from kubeconfig")
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
	}

	err = client.Get(context.Background(), runtimeclient.ObjectKey{
		Name:      sa.Name,
		Namespace: sa.Namespace,
	}, sa)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get service account")
	}

	if len(sa.Secrets) == 0 {
		return nil, nil
	}

	secretName := sa.Secrets[0].Name

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: istioOperatorNamespace,
		},
	}
	err = client.Get(context.Background(), runtimeclient.ObjectKey{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}, secret)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get secret")
	}

	clusterName := c.GetName()

	caData := secret.Data["ca.crt"]
	if !bytes.Contains(caData, config.CAData) {
		caData = append(append(caData, []byte("\n")...), config.CAData...)
	}

	yml := `apiVersion: v1
clusters:
   - cluster:
       certificate-authority-data: ` + base64.StdEncoding.EncodeToString(caData) + `
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
	return errors.WithStack(m.reconcileNamespace(istioOperatorNamespace, desiredState, c, nil))
}

func (m *MeshReconciler) reconcileRemoteIstioServiceAccount(desiredState DesiredState, c cluster.CommonCluster) error {
	resource := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
	}

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, resource))
	}

	return errors.WithStack(m.deleteResource(client, resource))
}

func (m *MeshReconciler) reconcileRemoteIstioClusterRole(desiredState DesiredState, c cluster.CommonCluster) error {
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

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, resource))
	}

	return errors.WithStack(m.deleteResource(client, resource))
}

func (m *MeshReconciler) reconcileRemoteIstioPrometheusService(desiredState DesiredState, c cluster.CommonCluster) error {
	resource := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-prometheus", c.GetName()),
			Namespace: backyardsNamespace,
			Labels: map[string]string{
				"backyards.banzaicloud.io/federated-prometheus": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http-admin",
					Port:       59090,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name":                fmt.Sprintf("%s-prometheus", backyardsReleaseName),
				"app.kubernetes.io/instance":            backyardsReleaseName,
				"backyards.banzaicloud.io/cluster-name": c.GetName(),
			},
		},
	}

	client, err := m.getRuntimeK8sClient(m.Master)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, resource))
	}

	return errors.WithStack(m.deleteResource(client, resource))
}

func (m *MeshReconciler) reconcileRemoteIstioClusterRoleBinding(desiredState DesiredState, c cluster.CommonCluster) error {
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

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, resource))
	}

	return errors.WithStack(m.deleteResource(client, resource))
}

func (m *MeshReconciler) reconcileRemoteIstioALSService(desiredState DesiredState, c cluster.CommonCluster) error {
	resource := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backyards-als",
			Namespace: backyardsNamespace,
			Labels: map[string]string{
				"app":                         "backyards-als",
				"app.kubernetes.io/component": "als",
				"app.kubernetes.io/instance":  "backyards",
				"app.kubernetes.io/name":      "backyards-als",
				"app.kubernetes.io/part-of":   "backyards",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("istio-pilot.%s.svc.cluster.local", istioOperatorNamespace),
		},
	}

	client, err := m.getRuntimeK8sClient(m.Master)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, resource))
	}

	return errors.WithStack(m.deleteResource(client, resource))
}

func (m *MeshReconciler) reconcileRemoteIstioTracingService(desiredState DesiredState, c cluster.CommonCluster) error {
	resource := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backyards-zipkin",
			Namespace: backyardsNamespace,
			Labels: map[string]string{
				"app":                         "backyards-als",
				"app.kubernetes.io/component": "tracing",
				"app.kubernetes.io/instance":  "backyards",
				"app.kubernetes.io/name":      "jaeger",
				"app.kubernetes.io/part-of":   "backyards",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("istio-pilot.%s.svc.cluster.local", istioOperatorNamespace),
		},
	}

	client, err := m.getRuntimeK8sClient(m.Master)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, resource))
	}

	return errors.WithStack(m.deleteResource(client, resource))
}

func (m *MeshReconciler) getRemoteClustersByExistingRemoteIstioCRs(c cluster.CommonCluster) (map[uint]cluster.CommonCluster, error) {
	clusters := make(map[uint]cluster.CommonCluster, 0)

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return nil, err
	}

	var remoteistios v1beta1.RemoteIstioList
	err = client.List(context.Background(), &remoteistios, runtimeclient.InNamespace(istioOperatorNamespace))
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, errors.WrapIf(err, "could not get remote istios")
	}

	for _, remoteistio := range remoteistios.Items {
		labels := remoteistio.GetLabels()
		if len(labels) == 0 {
			continue
		}
		cID := remoteistio.Labels[clusterIDLabel]
		if cID == "" {
			continue
		}

		clusterID, err := strconv.ParseUint(cID, 10, 64)
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		c, err := m.clusterGetter.GetClusterByID(context.Background(), c.GetOrganizationId(), uint(clusterID))
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		clusters[c.GetID()] = c.(cluster.CommonCluster)
	}

	return clusters, nil
}
