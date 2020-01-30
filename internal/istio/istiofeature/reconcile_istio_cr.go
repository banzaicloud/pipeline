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
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/istio-operator/pkg/apis/istio/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	istiooperatorclientset "github.com/banzaicloud/pipeline/.gen/istio-operator/pkg/client/clientset/versioned"

	"github.com/banzaicloud/pipeline/pkg/backoff"
)

func (m *MeshReconciler) ReconcileIstio(desiredState DesiredState) error {
	m.logger.Debug("reconciling Istio CR")
	defer m.logger.Debug("Istio CR reconciled")

	client, err := m.getMasterIstioOperatorK8sClient()
	if err != nil {
		return err
	}

	if desiredState == DesiredStatePresent {
		istio, err := client.IstioV1beta1().Istios(istioOperatorNamespace).Get(m.Configuration.name, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return errors.WrapIf(err, "could not check existence Istio CR")
		}

		if k8serrors.IsNotFound(err) {
			istio = &v1beta1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name: m.Configuration.name,
				},
			}
		}

		istio = m.configureIstioCR(istio, m.Configuration)

		if k8serrors.IsNotFound(err) {
			_, err = client.IstioV1beta1().Istios(istioOperatorNamespace).Create(istio)
			if err != nil {
				return errors.WrapIf(err, "could not create Istio CR")
			}
		} else if err == nil {
			_, err := client.IstioV1beta1().Istios(istioOperatorNamespace).Update(istio)
			if err != nil {
				return errors.WrapIf(err, "could not update Istio CR")
			}
		}
	} else {
		err := client.IstioV1beta1().Istios(istioOperatorNamespace).Delete(m.Configuration.name, &metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return errors.WrapIf(err, "could not remove Istio CR")
		}

		err = m.waitForIstioCRToBeDeleted(client)
		if err != nil {
			return errors.WrapIf(err, "timeout during waiting for Istio CR to be deleted")
		}
	}

	return nil
}

// waitForIstioCRToBeDeleted wait for Istio CR to be deleted
func (m *MeshReconciler) waitForIstioCRToBeDeleted(client *istiooperatorclientset.Clientset) error {
	m.logger.Debug("waiting for Istio CR to be deleted")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	err := backoff.Retry(func() error {
		_, err := client.IstioV1beta1().Istios(istioOperatorNamespace).Get(m.Configuration.name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}

		return errors.New("Istio CR still exists")
	}, backoffPolicy)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// configureIstioCR configures istio-operator specific CR based on the given params
func (m *MeshReconciler) configureIstioCR(istio *v1beta1.Istio, config Config) *v1beta1.Istio {
	enabled := true
	maxReplicas := int32(1)

	labels := istio.GetLabels()
	if labels == nil {
		labels = make(map[string]string, 0)
	}
	labels[clusterIDLabel] = strconv.FormatUint(uint64(m.Master.GetID()), 10)
	labels[cloudLabel] = m.Master.GetCloud()
	labels[distributionLabel] = m.Master.GetDistribution()
	istio.SetLabels(labels)

	istio.Spec.MTLS = config.EnableMTLS
	istio.Spec.AutoInjectionNamespaces = config.AutoSidecarInjectNamespaces
	istio.Spec.Version = istioVersion
	istio.Spec.ImagePullPolicy = corev1.PullAlways
	istio.Spec.Gateways.IngressConfig.MaxReplicas = 1
	istio.Spec.Gateways.EgressConfig.MaxReplicas = 1
	istio.Spec.Pilot = v1beta1.PilotConfiguration{
		Image:       m.Configuration.internalConfig.istioOperator.pilotImage,
		MaxReplicas: maxReplicas,
	}
	istio.Spec.Mixer = v1beta1.MixerConfiguration{
		K8sResourceConfiguration: v1beta1.K8sResourceConfiguration{
			Image:       &m.Configuration.internalConfig.istioOperator.mixerImage,
			MaxReplicas: &maxReplicas,
		},
	}
	istio.Spec.SidecarInjector.RewriteAppHTTPProbe = true
	istio.Spec.Tracing.Enabled = &enabled
	istio.Spec.Tracing.Zipkin.Address = zipkinAddress
	istio.Spec.Mixer.MultiClusterSupport = &enabled

	if len(m.Remotes) > 0 {
		istio.Spec.UseMCP = &enabled
		istio.Spec.MTLS = enabled
		istio.Spec.MeshExpansion = &enabled
		istio.Spec.ControlPlaneSecurityEnabled = enabled
	}

	if config.BypassEgressTraffic {
		istio.Spec.OutboundTrafficPolicy = v1beta1.OutboundTrafficPolicyConfiguration{
			Mode: "ALLOW_ANY",
		}
	} else {
		istio.Spec.OutboundTrafficPolicy = v1beta1.OutboundTrafficPolicyConfiguration{
			Mode: "REGISTRY_ONLY",
		}
	}

	return istio
}
