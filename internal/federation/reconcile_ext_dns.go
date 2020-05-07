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

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	internalhelm "github.com/banzaicloud/pipeline/internal/helm"

	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/src/dns"
	"github.com/banzaicloud/pipeline/src/helm"
)

func (m *FederationReconciler) ReconcileExternalDNSController(desiredState DesiredState) error {
	m.logger.Debug("start reconciling ExternalDNS controller")
	defer m.logger.Debug("finished reconciling ExternalDNS controller")

	infraNamespace := m.Configuration.dnsConfig.Namespace
	chartName := m.Configuration.dnsConfig.Charts.ExternalDNS.Chart
	const releaseName = "dns"

	_, err := EnsureCRDSourceForExtDNS(m.Host, infraNamespace, chartName, releaseName, desiredState, m.logger)
	if err != nil {
		return errors.WrapIf(err, "could not update ExternalDNS controller")
	}
	return nil
}

func EnsureCRDSourceForExtDNS(
	c internalhelm.ClusterDataProvider,
	namespace string,
	deploymentName string,
	releaseName string,
	desiredState DesiredState,
	logger logrus.FieldLogger,
) (upgraded bool, err error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return false, errors.WrapIf(err, "could not get k8s config")
	}

	hClient, err := pkgHelm.NewClient(kubeConfig, logger)
	if err != nil {
		return false, err
	}
	defer hClient.Close()

	resp, err := hClient.ReleaseContent(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Debug("externalDNS deployment not found")
			return false, nil
		}
		return false, err
	}

	crdPresent := false

	currentValues := &dns.ExternalDnsChartValues{}
	err = yaml.Unmarshal([]byte(resp.Release.Config.Raw), &currentValues)
	if err != nil {
		return false, err
	}
	if currentValues != nil {
		sources := currentValues.Sources
		for _, src := range sources {
			if src == "crd" {
				crdPresent = true
			}
		}
	}

	if desiredState == DesiredStatePresent && crdPresent {
		return false, nil
	}
	if desiredState == DesiredStateAbsent && !crdPresent {
		return false, nil
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
		return false, errors.WrapIf(err, "could not marshal chart value overrides")
	}

	_, err = helm.UpgradeDeployment(releaseName, deploymentName, resp.Release.Chart.Metadata.Version, nil, valuesOverride, true, kubeConfig, helm.GeneratePlatformHelmRepoEnv())
	if err != nil {
		return false, errors.WrapIfWithDetails(err, "could not upgrade deployment", "deploymentName", deploymentName)
	}
	return true, nil
}
