// Copyright © 2018 Banzai Cloud
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
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
)

// InstallMonitoring installs monitoring tools (Prometheus, Grafana) to a cluster.
func InstallMonitoring(cluster CommonCluster) error {
	monitoringNamespace := viper.GetString(pipConfig.PipelineSystemNamespace)

	clusterNameSecretTag := fmt.Sprintf("cluster:%s", cluster.GetName())
	clusterUidSecretTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())
	releaseSecretTag := fmt.Sprintf("release:%s", pipConfig.MonitorReleaseName)

	// Generating Grafana credentials
	grafanaAdminUsername := viper.GetString("monitor.grafanaAdminUsername")
	grafanaAdminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return emperror.Wrap(err, "failed to generate Grafana admin user password")
	}

	grafanaSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-grafana", cluster.GetID()),
		Type: pkgSecret.PasswordSecretType,
		Values: map[string]string{
			pkgSecret.Username: grafanaAdminUsername,
			pkgSecret.Password: grafanaAdminPass,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			pkgSecret.TagBanzaiReadonly,
			releaseSecretTag,
			"app:grafana",
		},
	}
	grafanaSecretID, err := secret.Store.CreateOrUpdate(cluster.GetOrganizationId(), &grafanaSecretRequest)
	if err != nil {
		return emperror.Wrap(err, "error store prometheus secret")
	}
	log.WithField("secretId", grafanaSecretID).Debugf("grafana secret stored")

	// Generating Prometheus credentials
	prometheusSecretName := fmt.Sprintf("cluster-%d-prometheus", cluster.GetID())

	// In order to regenerate a this secret we need to delete it first,
	// because updating will overwrite the htpasswd file in the secret
	_, err = secret.Store.GetByName(cluster.GetOrganizationId(), prometheusSecretName)
	if err != secret.ErrSecretNotExists {
		err := secret.Store.Delete(cluster.GetOrganizationId(), secret.GenerateSecretIDFromName(prometheusSecretName))
		if err != nil {
			return emperror.Wrap(err, "failed to regenerate prometheus credentials")
		}
	}

	prometheusAdminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return emperror.Wrap(err, "prometheus password generation failed")
	}

	prometheusSecretRequest := &secret.CreateSecretRequest{
		Name: prometheusSecretName,
		Type: pkgSecret.HtpasswdSecretType,
		Values: map[string]string{
			pkgSecret.Username: "prometheus",
			pkgSecret.Password: prometheusAdminPass,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			pkgSecret.TagBanzaiReadonly,
			releaseSecretTag,
		},
	}
	prometheusSecretID, err := secret.Store.CreateOrUpdate(cluster.GetOrganizationId(), prometheusSecretRequest)
	if err != nil {
		return emperror.Wrap(err, "error store prometheus secret")
	}
	log.WithField("secretId", prometheusSecretID).Debugf("prometheus secret stored")

	const kubePrometheusSecretName = "prometheus-basic-auth"

	installPromSecretRequest := InstallSecretRequest{
		SourceSecretName: prometheusSecretName,
		Namespace:        monitoringNamespace,
		Spec: map[string]InstallSecretRequestSpecItem{
			"auth": {Source: pkgSecret.HtpasswdFile},
		},
		Update: true,
	}
	prometheusK8Secret, err := InstallSecret(cluster, kubePrometheusSecretName, installPromSecretRequest)
	if err != nil {
		return emperror.Wrap(err, "failed to install tls secret to cluster")
	}
	log.Debugf("installed secret on cluster: %s", prometheusK8Secret)

	orgId := cluster.GetOrganizationId()
	org, err := auth.GetOrganizationById(orgId)
	if err != nil {
		return emperror.WrapWith(err, "failed to get organization", "organizationId", orgId)
	}

	baseDomain, err := dns.GetBaseDomain()
	if err != nil {
		return emperror.Wrap(err, "failed to get base domain")
	}

	host := strings.ToLower(fmt.Sprintf("%s.%s.%s", cluster.GetName(), org.Name, baseDomain))
	err = dns.ValidateSubdomain(host)
	if err != nil {
		return emperror.Wrap(err, "invalid grafana ingress host")
	}

	log.Debugf("grafana ingress host: %s", host)

	values := map[string]interface{}{
		"grafana": map[string]interface{}{
			"adminUser":     grafanaAdminUsername,
			"adminPassword": grafanaAdminPass,
			"ingress":       map[string][]string{"hosts": {host}},
			"affinity":      GetHeadNodeAffinity(cluster),
			"tolerations":   GetHeadNodeTolerations(),
		},
		"prometheus": map[string]interface{}{
			"alertmanager": map[string]interface{}{
				"affinity":    GetHeadNodeAffinity(cluster),
				"tolerations": GetHeadNodeTolerations(),
			},
			"kubeStateMetrics": map[string]interface{}{
				"affinity":    GetHeadNodeAffinity(cluster),
				"tolerations": GetHeadNodeTolerations(),
			},
			"nodeExporter": map[string]interface{}{
				"tolerations": GetHeadNodeTolerations(),
			},
			"server": map[string]interface{}{
				"affinity":    GetHeadNodeAffinity(cluster),
				"tolerations": GetHeadNodeTolerations(),
				"ingress": map[string]interface{}{
					"enabled": true,
					"annotations": map[string]string{
						"traefik.ingress.kubernetes.io/auth-type":   "basic",
						"traefik.ingress.kubernetes.io/auth-secret": kubePrometheusSecretName,
					},
					"hosts": []string{
						host + "/prometheus",
					},
				},
			},
			"pushgateway": map[string]interface{}{
				"affinity":    GetHeadNodeAffinity(cluster),
				"tolerations": GetHeadNodeTolerations(),
			},
		},
	}

	valuesJSON, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "values JSON conversion failed")
	}

	err = installDeployment(
		cluster,
		monitoringNamespace,
		pkgHelm.BanzaiRepository+"/pipeline-cluster-monitor",
		pipConfig.MonitorReleaseName,
		valuesJSON,
		"",
		false,
	)
	if err != nil {
		return emperror.Wrap(err, "install pipeline-cluster-monitor failed")
	}

	cluster.SetMonitoring(true)

	return nil
}
