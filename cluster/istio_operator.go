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

package cluster

import (
	"emperror.dev/emperror"
	"github.com/banzaicloud/istio-operator/pkg/apis/istio/v1beta1"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	pConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/istio"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

const istioOperatorNamespace = "istio-system"
const istioOperatorDeploymentName = pkgHelm.BanzaiRepository + "/" + "istio-operator"
const istioOperatorReleaseName = "istio-operator"

// installIstioOperator installs istio-operator on a cluster
func installIstioOperator(cluster CommonCluster) error {
	err := installDeployment(
		cluster,
		istioOperatorNamespace,
		istioOperatorDeploymentName,
		istioOperatorReleaseName,
		[]byte{},
		viper.GetString(pConfig.IstioOperatorChartVersion),
		true)
	if err != nil {
		return emperror.Wrap(err, "installing istio-operator with helm failed")
	}

	return nil
}

// createIstioCR creates an istio-operator specific CR which triggers the istio-operator to install Istio
func createIstioCR(kubeConfig []byte, params *InstallServiceMeshParams, cluster CommonCluster) error {
	restClient, err := createRESTClient(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create REST client")
	}

	istioConfig := createIstioConfig(params, cluster)

	err = restClient.Post().
		Namespace(istio.Namespace).
		Resource("istios").
		Body(&istioConfig).
		Do().
		Error()
	if err != nil {
		return emperror.Wrap(err, "failed to create Istio CR with RESTClient")
	}

	return nil
}

// createRESTClient creates a RESTClient to be able to operate on istio-operator specific CR
func createRESTClient(kubeConfig []byte) (restClient *rest.RESTClient, err error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create client from kubeconfig")
	}

	err = v1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to add istio-operator schema")
	}

	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "istio.banzaicloud.io", Version: "v1beta1"}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err = rest.RESTClientFor(config)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create REST client for config")
	}

	return restClient, nil
}

// createIstioConfig creates istio-operator specific CR based on the given posthook params
func createIstioConfig(params *InstallServiceMeshParams, cluster CommonCluster) v1beta1.Istio {
	istioConfig := v1beta1.Istio{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Istio",
			APIVersion: "istio.banzaicloud.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-config",
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
		},
		Spec: v1beta1.IstioSpec{
			MTLS:                    params.EnableMtls,
			AutoInjectionNamespaces: params.AutoSidecarInjectNamespaces,
		},
	}

	if params.BypassEgressTraffic {
		istioConfig.Spec.OutboundTrafficPolicy = v1beta1.OutboundTrafficPolicyConfiguration{
			Mode: "ALLOW_ANY",
		}
	} else {
		istioConfig.Spec.OutboundTrafficPolicy = v1beta1.OutboundTrafficPolicyConfiguration{
			Mode: "REGISTRY_ONLY",
		}
	}

	return istioConfig
}
