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

	"emperror.dev/emperror"
	"github.com/ghodss/yaml"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/secret"
)

const MonitorReleaseName = "monitor"

// InstallMonitoring installs monitoring tools (Prometheus, Grafana) to a cluster.
// Deprecated: use monitoring feature instead.
func InstallMonitoring(cluster CommonCluster) error {
	monitoringNamespace := global.Config.Cluster.Namespace

	clusterNameSecretTag := fmt.Sprintf("cluster:%s", cluster.GetName())
	clusterUidSecretTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())
	releaseSecretTag := fmt.Sprintf("release:%s", MonitorReleaseName)

	// Generating Grafana credentials
	grafanaAdminUsername := global.Config.Cluster.Monitoring.Grafana.AdminUser
	grafanaAdminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return emperror.Wrap(err, "failed to generate Grafana admin user password")
	}

	grafanaSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-grafana", cluster.GetID()),
		Type: secrettype.PasswordSecretType,
		Values: map[string]string{
			secrettype.Username: grafanaAdminUsername,
			secrettype.Password: grafanaAdminPass,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			secret.TagBanzaiReadonly,
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
		Type: secrettype.HtpasswdSecretType,
		Values: map[string]string{
			secrettype.Username: "prometheus",
			secrettype.Password: prometheusAdminPass,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			secret.TagBanzaiReadonly,
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
			"auth": {Source: secrettype.HtpasswdFile},
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

	var host string

	if global.Config.Cluster.DNS.BaseDomain != "" {
		baseDomain, err := dns.GetBaseDomain()
		if err != nil {
			return emperror.Wrap(err, "failed to get base domain")
		}

		host = strings.ToLower(fmt.Sprintf("%s.%s.%s", cluster.GetName(), org.Name, baseDomain))
		err = dns.ValidateSubdomain(host)
		if err != nil {
			return emperror.Wrap(err, "invalid grafana ingress host")
		}
	}

	log.Debugf("grafana ingress host: %s", host)

	values := map[string]interface{}{
		"grafana": map[string]interface{}{
			"adminUser":     grafanaAdminUsername,
			"adminPassword": grafanaAdminPass,
			"ingress":       map[string][]string{"hosts": {host}},
		},
		"prometheus": map[string]interface{}{
			"server": map[string]interface{}{
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
		MonitorReleaseName,
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
