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
	"fmt"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/istio-operator/pkg/apis/istio/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) ReconcileRemoteIstio(desiredState DesiredState, c cluster.CommonCluster) error {
	m.logger.Debug("reconciling Remote Istio CR")
	defer m.logger.Debug("Remote Istio CR reconciled")

	remoteIstio := m.generateRemoteIstioCR(m.Configuration, c)

	client, err := m.getRuntimeK8sClient(m.Master)
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, &remoteIstio))
	}

	err = m.deleteResource(client, &remoteIstio)
	if err != nil {
		return errors.WithStack(err)
	}

	err = m.waitForRemoteIstioCRToBeDeleted(c.GetName(), client)
	if err != nil {
		return errors.WrapIf(err, "timeout during waiting for Remote Istio CR to be deleted")
	}

	return nil
}

// waitForRemoteIstioCRToBeDeleted wait for Remote Istio CR to be deleted
func (m *MeshReconciler) waitForRemoteIstioCRToBeDeleted(name string, client client.Client) error {
	m.logger.WithField("name", name).Debug("waiting for Remote Istio CR to be deleted")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	err := backoff.Retry(func() error {
		var remoteIstio v1beta1.RemoteIstio
		err := client.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: istioOperatorNamespace,
		}, &remoteIstio)
		if k8serrors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return errors.WrapIfWithDetails(err, "could not check Remote Istio CR existence", "name", name)
		}

		return errors.NewWithDetails("Remote Istio CR still exists", "name", name)
	}, backoffPolicy)

	return err
}

// generateRemoteIstioCR generates istio-operator specific CR based on the given params
func (m *MeshReconciler) generateRemoteIstioCR(config Config, c cluster.CommonCluster) v1beta1.RemoteIstio {
	enabled := true
	replicaCount := int32(1)

	istioConfig := v1beta1.RemoteIstio{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetName(),
			Namespace: istioOperatorNamespace,
			Labels: map[string]string{
				clusterIDLabel:    strconv.FormatUint(uint64(c.GetID()), 10),
				cloudLabel:        c.GetCloud(),
				distributionLabel: c.GetDistribution(),
			},
		},
		Spec: v1beta1.RemoteIstioSpec{
			AutoInjectionNamespaces: config.AutoSidecarInjectNamespaces,
			Citadel: v1beta1.CitadelConfiguration{
				Enabled: &enabled,
			},
			EnabledServices: []v1beta1.IstioService{
				{
					Name: "istio-pilot",
					Ports: []corev1.ServicePort{
						{
							Port:     65000,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
				{
					Name: "istio-policy",
					Ports: []corev1.ServicePort{
						{
							Port:     65000,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
				{
					Name: "istio-telemetry",
					Ports: []corev1.ServicePort{
						{
							Port:     65000,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	istioConfig.Spec.SidecarInjector.Enabled = &enabled
	istioConfig.Spec.SidecarInjector.ReplicaCount = &replicaCount
	istioConfig.Spec.SidecarInjector.InjectedContainerAdditionalEnvVars = []corev1.EnvVar{
		{
			Name:  "ISTIO_METAJSON_PLATFORM_METADATA",
			Value: fmt.Sprintf(`{"PLATFORM_METADATA":{"cluster_id":"%s"}}`, c.GetName()),
		},
	}
	return istioConfig
}
