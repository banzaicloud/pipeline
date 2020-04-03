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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) ReconcileIstio(desiredState DesiredState, c cluster.CommonCluster) error {
	m.logger.Debug("reconciling Istio CR")
	defer m.logger.Debug("Istio CR reconciled")

	client, err := m.getRuntimeK8sClient(c)
	if err != nil {
		return errors.WithStack(err)
	}

	err = m.waitForCRD(v1beta1.Resource("istios").String(), client)
	if err != nil {
		return errors.WrapIf(err, "error while waiting for Istio CRD")
	}

	image := m.Configuration.internalConfig.Charts.IstioOperator.Values.Operator.Image
	imageWithTag := fmt.Sprintf("%s:%s", image.Repository, image.Tag)

	err = m.waitForPod(client, istioOperatorNamespace, map[string]string{"app.kubernetes.io/instance": "istio-operator"}, imageWithTag)
	if err != nil {
		return errors.WrapIf(err, "error while waiting for Istio operator pod")
	}

	istio := &v1beta1.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Configuration.name,
			Namespace: istioOperatorNamespace,
		},
	}
	m.configureIstioCR(istio, m.Configuration)

	if desiredState == DesiredStatePresent {
		return errors.WithStack(m.applyResource(client, istio))
	}

	err = m.deleteResource(client, istio)
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WrapIf(m.waitForIstioCRToBeDeleted(client), "timeout during waiting for Istio CR to be deleted")
}

// waitForIstioCRToBeDeleted wait for Istio CR to be deleted
func (m *MeshReconciler) waitForIstioCRToBeDeleted(client client.Client) error {
	m.logger.Debug("waiting for Istio CR to be deleted")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	err := backoff.Retry(func() error {
		var istio v1beta1.Istio
		err := client.Get(context.Background(), types.NamespacedName{
			Name:      m.Configuration.name,
			Namespace: istioOperatorNamespace,
		}, &istio)
		if k8serrors.IsNotFound(err) {
			return nil
		}

		return errors.New("Istio CR still exists")
	}, backoffPolicy)

	return errors.WithStack(err)
}

// configureIstioCR configures istio-operator specific CR based on the given params
func (m *MeshReconciler) configureIstioCR(istio *v1beta1.Istio, config Config) {
	enabled := true
	disabled := false
	maxReplicas := int32(1)

	labels := istio.GetLabels()
	if labels == nil {
		labels = make(map[string]string, 0)
	}
	labels[clusterIDLabel] = strconv.FormatUint(uint64(m.Master.GetID()), 10)
	labels[cloudLabel] = m.Master.GetCloud()
	labels[distributionLabel] = m.Master.GetDistribution()
	istio.SetLabels(labels)

	istio.Spec.Gateways.IngressConfig.Ports = []corev1.ServicePort{
		{Name: "status-port", Port: 15020, TargetPort: intstr.FromInt(15020)},
		{Name: "http2", Port: 80, TargetPort: intstr.FromInt(80)},
		{Name: "https", Port: 443, TargetPort: intstr.FromInt(443)},
		{Name: "tls", Port: 15443, TargetPort: intstr.FromInt(15443)},
		{Name: "tcp-als-tls", Port: 50600, TargetPort: intstr.FromInt(50600)},
		{Name: "tcp-zipkin-tls", Port: 59411, TargetPort: intstr.FromInt(59411)},
	}

	istio.Spec.MTLS = nil
	if config.EnableMTLS {
		istio.Spec.MeshPolicy.MTLSMode = "PERMISSIVE"
	} else {
		istio.Spec.MeshPolicy.MTLSMode = "DISABLED"
	}
	istio.Spec.AutoMTLS = &enabled
	istio.Spec.AutoInjectionNamespaces = config.AutoSidecarInjectNamespaces
	istio.Spec.Version = istioVersion
	istio.Spec.ImagePullPolicy = corev1.PullAlways
	istio.Spec.Gateways.IngressConfig.Enabled = &enabled
	istio.Spec.Gateways.IngressConfig.MaxReplicas = &maxReplicas
	istio.Spec.Gateways.EgressConfig.Enabled = &enabled
	istio.Spec.Gateways.EgressConfig.MaxReplicas = &maxReplicas
	istio.Spec.Pilot.Enabled = &disabled
	istio.Spec.Pilot.Image = &m.Configuration.internalConfig.Istio.PilotImage
	istio.Spec.Pilot.MaxReplicas = &maxReplicas
	istio.Spec.Mixer.Enabled = &disabled
	istio.Spec.Mixer.MultiClusterSupport = &enabled
	istio.Spec.Telemetry.Enabled = &disabled
	istio.Spec.Policy.Enabled = &disabled
	istio.Spec.Galley.Enabled = &disabled
	istio.Spec.Citadel.Enabled = &disabled
	istio.Spec.Istiod.Enabled = &enabled
	istio.Spec.Istiod.MultiClusterSupport = &enabled
	istio.Spec.Mixer.Image = &m.Configuration.internalConfig.Istio.MixerImage
	istio.Spec.Mixer.MaxReplicas = &maxReplicas
	istio.Spec.SidecarInjector.Enabled = &disabled
	istio.Spec.SidecarInjector.Image = &m.Configuration.internalConfig.Istio.SidecarInjectorImage
	istio.Spec.SidecarInjector.RewriteAppHTTPProbe = true
	istio.Spec.SidecarInjector.InjectedContainerAdditionalEnvVars = []corev1.EnvVar{
		{
			Name:  "ISTIO_METAJSON_PLATFORM_METADATA",
			Value: `{"PLATFORM_METADATA":{"cluster_id":"master"}}`,
		},
	}
	istio.Spec.Tracing.Enabled = &enabled
	istio.Spec.Tracing.Zipkin = v1beta1.ZipkinConfiguration{
		Address: fmt.Sprintf("%s:%d", zipkinHost, zipkinPort),
		TLSSettings: &v1beta1.TLSSettings{
			Mode: "ISTIO_MUTUAL",
		},
	}

	istio.Spec.Proxy.Image = m.Configuration.internalConfig.Istio.ProxyImage
	istio.Spec.Proxy.EnvoyAccessLogService = v1beta1.EnvoyServiceCommonConfiguration{
		Enabled: &enabled,
		Host:    alsHost,
		Port:    alsPort,
		TLSSettings: &v1beta1.TLSSettings{
			Mode: "ISTIO_MUTUAL",
		},
		TCPKeepalive: &v1beta1.TCPKeepalive{
			Interval: "10s",
			Probes:   3,
			Time:     "10s",
		},
	}
	istio.Spec.Proxy.UseMetadataExchangeFilter = &enabled
	istio.Spec.JWTPolicy = "first-party-jwt"
	istio.Spec.ControlPlaneSecurityEnabled = enabled
	istio.Spec.MixerlessTelemetry = &v1beta1.MixerlessTelemetryConfiguration{
		Enabled: &enabled,
	}

	if len(m.Remotes) > 0 {
		istio.Spec.Gateways.IngressConfig.Labels = map[string]string{"istio.banzaicloud.io/mesh-expansion": "true"}
		istio.Spec.MeshExpansion = &enabled
		istio.Spec.MeshPolicy.MTLSMode = "PERMISSIVE"
	} else {
		istio.Spec.Gateways.IngressConfig.Labels = nil
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
}
