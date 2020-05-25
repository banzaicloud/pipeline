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
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"

	internalhelm "github.com/banzaicloud/pipeline/internal/helm"

	"github.com/banzaicloud/pipeline/src/dns"
)

func (m *FederationReconciler) ReconcileExternalDNSController(desiredState DesiredState) error {
	m.logger.Debug("start reconciling ExternalDNS controller")
	defer m.logger.Debug("finished reconciling ExternalDNS controller")

	infraNamespace := m.Configuration.dnsConfig.Namespace
	chartName := m.Configuration.dnsConfig.Charts.ExternalDNS.Chart
	const releaseName = "dns"

	_, err := EnsureCRDSourceForExtDNS(m.Host, m.helmService, infraNamespace, chartName, releaseName, desiredState, m.logger)
	if err != nil {
		return errors.WrapIf(err, "could not update ExternalDNS controller")
	}
	return nil
}

func EnsureCRDSourceForExtDNS(
	c internalhelm.ClusterDataProvider,
	helmService HelmService,
	namespace string,
	deploymentName string,
	releaseName string,
	desiredState DesiredState,
	logger logrus.FieldLogger,
) (upgraded bool, err error) {
	resp, err := helmService.GetRelease(c, releaseName, namespace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Debug("externalDNS deployment not found")
			return false, nil
		}
		return false, err
	}

	crdPresent := false

	currentValues := &dns.ExternalDnsChartValues{}
	err = mapstructure.Decode(&resp.ReleaseInfo.Values, currentValues)
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

	var valuesOverride map[string]interface{}
	err = mapstructure.Decode(&values, &valuesOverride)
	if err != nil {
		return false, errors.WrapIf(err, "could not decode chart value overrides")
	}

	err = helmService.InstallOrUpgrade(
		c,
		internalhelm.Release{
			ReleaseName: releaseName,
			ChartName:   deploymentName,
			Namespace:   namespace,
			Values:      valuesOverride,
			Version:     resp.Version,
		}, internalhelm.Options{
			Namespace:   namespace,
			ReuseValues: true,
			Install:     true,
		},
	)
	if err != nil {
		return false, errors.WrapIfWithDetails(err, "could not upgrade deployment", "deploymentName", deploymentName)
	}
	return true, nil
}
