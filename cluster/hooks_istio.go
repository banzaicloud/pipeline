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
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/istio"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// InstallServiceMeshParams describes InstallServiceMesh posthook params
type InstallServiceMeshParams struct {
	// AutoSidecarInjectNamespaces list of namespaces that will be labelled with istio-injection=enabled
	AutoSidecarInjectNamespaces []string `json:"autoSidecarInjectNamespaces,omitempty"`
	// BypassEgressTraffic prevents Envoy sidecars from intercepting external requests
	BypassEgressTraffic bool `json:"bypassEgressTraffic,omitempty"`
	// EnableMtls signals if mutual TLS is enabled in the service mesh
	EnableMtls bool `json:"mtls,omitempty"`
}

// InstallServiceMesh is a posthook for installing Istio on a cluster
func InstallServiceMesh(cluster CommonCluster, param cluster.PostHookParam) error {
	var params InstallServiceMeshParams
	err := castToPostHookParam(&param, &params)
	if err != nil {
		return emperror.Wrap(err, "failed to cast posthook param")
	}

	log := log.WithFields(logrus.Fields{"cluster": cluster.GetName(), "clusterID": cluster.GetID()})

	log.Infof("istio params: %#v", params)

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "failed to get kubeconfig")
	}

	err = installIstioOperator(cluster)
	if err != nil {
		return emperror.Wrap(err, "failed to create istio-operator")
	}

	err = createIstioCR(kubeConfig, &params, cluster)
	if err != nil {
		return emperror.Wrap(err, "failed to create Istio CR")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create client from kubeconfig")
	}
	if cluster.GetMonitoring() {
		err = istio.AddPrometheusTargets(log, client)
		if err != nil {
			return emperror.Wrap(err, "failed to add prometheus targets")
		}
		err = istio.AddGrafanaDashboards(log, client)
		if err != nil {
			return emperror.Wrap(err, "failed to add grafana dashboards")
		}
	}

	cluster.SetServiceMesh(true)
	return nil
}
