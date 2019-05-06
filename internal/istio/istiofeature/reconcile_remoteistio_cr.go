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
	"strings"
	"time"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/istio-operator/pkg/apis/istio/v1beta1"
	istiooperatorclientset "github.com/banzaicloud/istio-operator/pkg/client/clientset/versioned"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/backoff"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

func (m *MeshReconciler) ReconcileRemoteIstio(desiredState DesiredState, c cluster.CommonCluster) error {
	m.logger.Debug("reconciling Remote Istio CR")
	defer m.logger.Debug("Remote Istio CR reconciled")

	client, err := m.GetMasterIstioOperatorK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		_, err := client.IstioV1beta1().RemoteIstios(istioOperatorNamespace).Get(c.GetName(), metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return emperror.Wrap(err, "could not check existance Remote Istio CR")
		}

		if err == nil {
			m.logger.Debug("Remote Istio CR already exists")
			return nil
		}

		ipRanges, err := c.GetK8sIpv4Cidrs()
		if err != nil {
			return emperror.Wrap(err, "could not get ipv4 ranges for cluster")
		}
		remoteIstioCR := m.generateRemoteIstioCR(m.Configuration, ipRanges, c)
		_, err = client.IstioV1beta1().RemoteIstios(istioOperatorNamespace).Create(&remoteIstioCR)
		if err != nil {
			return emperror.Wrap(err, "could not create Istio CR")
		}
	} else {
		err := client.IstioV1beta1().RemoteIstios(istioOperatorNamespace).Delete(c.GetName(), &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return emperror.Wrap(err, "could not remove Istio CR")
		}

		err = m.waitForRemoteIstioCRToBeDeleted(c.GetName(), client)
		if err != nil {
			return emperror.Wrap(err, "timeout during waiting for Remote Istio CR to be deleted")
		}
	}

	return nil
}

// waitForRemoteIstioCRToBeDeleted wait for Istio CR to be deleted
func (m *MeshReconciler) waitForRemoteIstioCRToBeDeleted(name string, client *istiooperatorclientset.Clientset) error {
	m.logger.WithField("name", name).Debug("waiting for Remote Istio CR to be deleted")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	err := backoff.Retry(func() error {
		_, err := client.IstioV1beta1().RemoteIstios(istioOperatorNamespace).Get(name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return emperror.WrapWith(err, "could not check Remote Istio CR existance", "name", name)
		}

		return emperror.With(errors.New("Remote Istio CR still exists"), "name", name)
	}, backoffPolicy)

	return err
}

// generateRemoteIstioCR generates istio-operator specific CR based on the given params
func (m *MeshReconciler) generateRemoteIstioCR(config Config, ipRanges *pkgCluster.Ipv4Cidrs, c cluster.CommonCluster) v1beta1.RemoteIstio {
	enabled := true
	istioConfig := v1beta1.RemoteIstio{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.GetName(),
			Labels: map[string]string{
				"controller-tools.k8s.io":              "1.0",
				"cluster.banzaicloud.com/id":           strconv.FormatUint(uint64(c.GetID()), 10),
				"cluster.banzaicloud.com/cloud":        c.GetCloud(),
				"cluster.banzaicloud.com/distribution": c.GetDistribution(),
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
			SidecarInjector: v1beta1.SidecarInjectorConfiguration{
				Enabled:      &enabled,
				ReplicaCount: 1,
			},
		},
	}

	if config.BypassEgressTraffic {
		istioConfig.Spec.IncludeIPRanges = strings.Join(ipRanges.PodIPRanges, ",") + "," + strings.Join(ipRanges.ServiceClusterIPRanges, ",")
	}

	return istioConfig
}
