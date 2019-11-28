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
	"fmt"
	"strings"

	"emperror.dev/emperror"
	"github.com/ghodss/yaml"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/global"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/src/dns"
	"github.com/banzaicloud/pipeline/src/helm"
)

func (m *FederationReconciler) ReconcileExternalDNSController(desiredState DesiredState) error {
	m.logger.Debug("start reconciling ExternalDNS controller")
	defer m.logger.Debug("finished reconciling ExternalDNS controller")

	infraNamespace := global.Config.Cluster.DNS.Namespace
	chartName := global.Config.Cluster.DNS.Charts.ExternalDNS.Chart
	const releaseName = "dns"

	err := m.ensureCRDSourceForExtDNS(m.Host, infraNamespace, chartName, releaseName, desiredState)
	if err != nil {
		return emperror.Wrap(err, "could not update ExternalDNS controller")
	}
	return nil
}

func (m *FederationReconciler) ensureCRDSourceForExtDNS(
	c cluster.CommonCluster,
	namespace string,
	deploymentName string,
	releaseName string,
	desiredState DesiredState,
) error {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "could not get k8s config")
	}

	org, err := auth.GetOrganizationById(c.GetOrganizationId())
	if err != nil {
		return emperror.Wrap(err, "could not get organization")
	}

	hClient, err := pkgHelm.NewClient(kubeConfig, m.logger)
	if err != nil {

		return err
	}
	defer hClient.Close()

	resp, err := hClient.ReleaseContent(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.logger.Debug("externalDNS deployment not found")
			return nil
		}
		return err
	}

	crdPresent := false

	currentValues := &dns.ExternalDnsChartValues{}
	err = yaml.Unmarshal([]byte(resp.Release.Config.Raw), &currentValues)
	if err != nil {
		return err
	}
	sources := currentValues.Sources
	for _, src := range sources {
		if src == "crd" {
			crdPresent = true
		}
	}

	if desiredState == DesiredStatePresent && crdPresent {
		return nil
	}
	if desiredState == DesiredStateAbsent && !crdPresent {
		return nil
	}

	values := dns.ExternalDnsChartValues{
		Sources: []string{
			"service",
			"ingress",
			"crd",
		},
		ExtraArgs: map[string]string{
			"crd-source-apiversion": fmt.Sprintf("%s/%s", multiClusterGroup, multiClusterGroupVersion),
			"crd-source-kind":       "DNSEndpoint",
		},
		TxtPrefix: "cname",
	}

	if desiredState == DesiredStateAbsent {
		values = dns.ExternalDnsChartValues{
			Sources: []string{
				"service",
				"ingress",
			},
			ExtraArgs: map[string]string{},
		}
	}
	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	_, err = helm.UpgradeDeployment(releaseName, deploymentName, resp.Release.Chart.Metadata.Version, nil, valuesOverride, true, kubeConfig, helm.GenerateHelmRepoEnv(org.Name))
	if err != nil {
		return emperror.WrapWith(err, "could not upgrade deployment", "deploymentName", deploymentName)
	}
	return nil
}
