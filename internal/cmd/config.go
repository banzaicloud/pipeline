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

package cmd

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterconfig"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/logging"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/monitoring"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/vault"
)

// AuthOIDCConfig contains OIDC auth configuration.
type AuthOIDCConfig struct {
	Issuer       string
	Insecure     bool
	ClientID     string
	ClientSecret string
}

// Validate validates the configuration.
func (c AuthOIDCConfig) Validate() error {
	if c.Issuer == "" {
		return errors.New("auth oidc issuer is required")
	}

	if c.ClientID == "" {
		return errors.New("auth oidc client ID is required")
	}

	if c.ClientSecret == "" {
		return errors.New("auth oidc client secret is required")
	}

	return nil
}

// AuthTokenConfig contains auth configuration.
type AuthTokenConfig struct {
	SigningKey string
	Issuer     string
	Audience   string
}

// Validate validates the configuration.
func (c AuthTokenConfig) Validate() error {
	if c.SigningKey == "" {
		return errors.New("auth token signing key is required")
	}

	if len(c.SigningKey) < 32 {
		return errors.New("auth token signing key must be at least 32 characters")
	}

	if c.Issuer == "" {
		return errors.New("auth token issuer is required")
	}

	if c.Audience == "" {
		return errors.New("auth token audience is required")
	}

	return nil
}

// ClusterConfig contains cluster configuration.
type ClusterConfig struct {
	// Initial manifest
	Manifest string

	// Namespace to install Pipeline components to
	Namespace string

	Labels clusterconfig.LabelConfig

	// Features
	Vault        ClusterVaultConfig
	Monitoring   ClusterMonitoringConfig
	Logging      ClusterLoggingConfig
	DNS          ClusterDNSConfig
	SecurityScan ClusterSecurityScanConfig
	Expiry       ExpiryConfig
}

// Validate validates the configuration.
func (c ClusterConfig) Validate() error {
	if c.Manifest != "" {
		file, err := os.OpenFile(c.Manifest, os.O_RDONLY, 0666)
		if err != nil {
			return fmt.Errorf("cluster manifest file is not readable: %w", err)
		}
		_ = file.Close()
	}

	if c.Namespace == "" {
		return errors.New("cluster namespace is required")
	}

	if err := c.Labels.Validate(); err != nil {
		return err
	}

	if c.Vault.Enabled {
		if err := c.Vault.Validate(); err != nil {
			return err
		}
	}

	if c.Monitoring.Enabled {
		if err := c.Monitoring.Validate(); err != nil {
			return err
		}
	}

	if c.Logging.Enabled {
		if err := c.Logging.Validate(); err != nil {
			return err
		}
	}

	if c.DNS.Enabled {
		if err := c.DNS.Validate(); err != nil {
			return err
		}
	}

	if c.SecurityScan.Enabled {
		if err := c.SecurityScan.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Process post-processes the configuration after loading (before validation).
func (c *ClusterConfig) Process() error {
	if c.Labels.Namespace == "" {
		c.Labels.Namespace = c.Namespace
	}

	if c.Vault.Namespace == "" {
		c.Vault.Namespace = c.Namespace
	}

	if c.Monitoring.Namespace == "" {
		c.Monitoring.Namespace = c.Namespace
	}

	if c.Logging.Namespace == "" {
		c.Logging.Namespace = c.Namespace
	}

	if c.DNS.Namespace == "" {
		c.DNS.Namespace = c.Namespace
	}

	return nil
}

// ClusterVaultConfig contains cluster vault configuration.
type ClusterVaultConfig struct {
	Enabled bool

	vault.Config `mapstructure:",squash"`
}

// ClusterMonitoringConfig contains cluster monitoring configuration.
type ClusterMonitoringConfig struct {
	Enabled bool

	monitoring.Config `mapstructure:",squash"`
}

// ClusterLoggingConfig contains cluster logging configuration.
type ClusterLoggingConfig struct {
	Enabled bool

	logging.Config `mapstructure:",squash"`
}

// ClusterDNSConfig contains cluster DNS configuration.
type ClusterDNSConfig struct {
	Enabled bool

	dns.Config `mapstructure:",squash"`
}

// ClusterSecurityScanConfig contains cluster security scan configuration.
type ClusterSecurityScanConfig struct {
	Enabled bool
	Anchore ClusterSecurityScanAnchoreConfig
}

func (c ClusterSecurityScanConfig) Validate() error {
	if c.Anchore.Enabled {
		if err := c.Anchore.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ClusterSecurityScanAnchoreConfig contains cluster security scan anchore configuration.
type ClusterSecurityScanAnchoreConfig struct {
	Enabled bool

	anchore.Config `mapstructure:",squash"`
}

func (c ClusterSecurityScanAnchoreConfig) Validate() error {
	if c.Enabled {
		if _, err := url.Parse(c.Endpoint); err != nil {
			return errors.Wrap(err, "anchore endpoint must be a valid URL")
		}

		if c.User == "" {
			return errors.New("anchore user is required")
		}

		if c.Password == "" {
			return errors.New("anchore password is required")
		}
	}

	return nil
}

type ExpiryConfig struct {
	Enabled bool
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, _ *pflag.FlagSet) {
	// Log configuration
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	v.SetDefault("log::format", "logfmt")
	v.SetDefault("log::level", "info")
	v.RegisterAlias("log::noColor", "no_color")

	// ErrorHandler configuration
	v.SetDefault("errors::stackdriver::enabled", false)
	v.SetDefault("errors::stackdriver::projectId", false)

	// Pipeline configuration
	v.SetDefault("pipeline::uuid", "")
	v.SetDefault("pipeline::external::url", "")
	v.SetDefault("pipeline::external::insecure", false)

	// Auth configuration
	v.SetDefault("auth::oidc::issuer", "")
	v.SetDefault("auth::oidc::insecure", false)
	v.SetDefault("auth::oidc::clientId", "")
	v.SetDefault("auth::oidc::clientSecret", "")

	v.SetDefault("auth::cli::clientId", "banzai-cli")

	v.SetDefault("auth::cookie::secure", true)
	v.SetDefault("auth::cookie::domain", "")
	v.SetDefault("auth::cookie::setDomain", false)

	v.SetDefault("auth::token::signingKey", "")
	v.SetDefault("auth::token::issuer", "")
	v.SetDefault("auth::token::audience", "")

	// Dex configuration
	v.SetDefault("dex::apiAddr", "")
	v.SetDefault("dex::apiCa", "")

	// Kubernetes configuration
	v.SetDefault("kubernetes::client::forceGlobal", false)

	// Database config
	v.SetDefault("database::dialect", "mysql")
	v.SetDefault("database::host", "")
	v.SetDefault("database::port", 3306)
	v.SetDefault("database::tls", "")
	v.SetDefault("database::user", "")
	v.SetDefault("database::password", "")
	v.SetDefault("database::name", "pipeline")
	v.SetDefault("database::params", map[string]string{
		"charset": "utf8mb4",
	})
	v.SetDefault("database::queryLog", false)

	// Cadence configuration
	v.SetDefault("cadence::host", "")
	v.SetDefault("cadence::port", 7933)
	v.SetDefault("cadence::domain", "pipeline")

	// Cluster configuration
	v.SetDefault("cluster::manifest", "")
	v.SetDefault("cluster::namespace", "pipeline-system")

	v.SetDefault("cluster::ingress::cert::source", "file")
	v.SetDefault("cluster::ingress::cert::path", "config/certs")

	v.SetDefault("cluster::labels::domain", "banzaicloud.io")
	v.SetDefault("cluster::labels::forbiddenDomains", []string{
		"k8s.io",
		"google.com",
		"coreos.com",
		"oraclecloud.com",
		"node.info",
		"azure.com",
		"agentpool",
		"storageprofile",
		"storagetier",
	})
	v.SetDefault("cluster::labels::charts::nodepoolLabelOperator::chart", "banzaicloud-stable/nodepool-labels-operator")
	v.SetDefault("cluster::labels::charts::nodepoolLabelOperator::version", "0.0.3")
	v.SetDefault("cluster::labels::charts::nodepoolLabelOperator::values", map[string]interface{}{})

	v.SetDefault("cluster::vault::enabled", true)
	v.SetDefault("cluster::vault::namespace", "")
	v.SetDefault("cluster::vault::managed::enabled", false)
	v.SetDefault("cluster::vault::managed::endpoint", "")
	v.SetDefault("cluster::vault::charts::webhook::chart", "banzaicloud-stable/vault-secrets-webhook")
	v.SetDefault("cluster::vault::charts::webhook::version", "0.6.0")
	v.SetDefault("cluster::vault::charts::webhook::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "banzaicloud/vault-secrets-webhook",
			"tag":        "0.6.0",
		},
	})

	v.SetDefault("cluster::monitoring::enabled", true)
	v.SetDefault("cluster::monitoring::namespace", "")
	v.SetDefault("cluster::monitoring::grafana::adminUser", "admin")
	v.SetDefault("cluster::monitoring::charts::operator::chart", "stable/prometheus-operator")
	v.SetDefault("cluster::monitoring::charts::operator::version", "7.2.0")
	v.SetDefault("cluster::monitoring::charts::operator::values", map[string]interface{}{
		"prometheus": map[string]interface{}{
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"traefik.frontend.rule.type":                 "PathPrefix",
					"traefik.ingress.kubernetes.io/ssl-redirect": "true",
				},
			},
		},
		"alertmanager": map[string]interface{}{
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"traefik.frontend.rule.type":                 "PathPrefix",
					"traefik.ingress.kubernetes.io/ssl-redirect": "true",
				},
			},
		},
		"grafana": map[string]interface{}{
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"traefik.frontend.rule.type":                         "PathPrefixStrip",
					"traefik.ingress.kubernetes.io/redirect-permanent":   "true",
					"traefik.ingress.kubernetes.io/redirect-regex":       "^http://(.*)",
					"traefik.ingress.kubernetes.io/redirect-replacement": `https://$1\`,
				},
			},
		},
	})
	v.SetDefault("cluster::monitoring::images::operator::repository", "quay.io/coreos/prometheus-operator")
	v.SetDefault("cluster::monitoring::images::operator::tag", "v0.32.0")
	v.SetDefault("cluster::monitoring::images::prometheus::repository", "quay.io/prometheus/prometheus")
	v.SetDefault("cluster::monitoring::images::prometheus::tag", "v2.12.0")
	v.SetDefault("cluster::monitoring::images::alertmanager::repository", "quay.io/prometheus/alertmanager")
	v.SetDefault("cluster::monitoring::images::alertmanager::tag", "v0.19.0")
	v.SetDefault("cluster::monitoring::images::grafana::repository", "grafana/grafana")
	v.SetDefault("cluster::monitoring::images::grafana::tag", "6.4.2")
	v.SetDefault("cluster::monitoring::images::kubestatemetrics::repository", "quay.io/coreos/kube-state-metrics")
	v.SetDefault("cluster::monitoring::images::kubestatemetrics::tag", "v1.8.0")
	v.SetDefault("cluster::monitoring::images::nodeexporter::repository", "quay.io/prometheus/node-exporter")
	v.SetDefault("cluster::monitoring::images::nodeexporter::tag", "v0.18.0")

	v.SetDefault("cluster::monitoring::charts::pushgateway::chart", "stable/prometheus-pushgateway")
	v.SetDefault("cluster::monitoring::charts::pushgateway::version", "1.0.1")
	v.SetDefault("cluster::monitoring::charts::pushgateway::values", map[string]interface{}{})
	v.SetDefault("cluster::monitoring::images::pushgateway::repository", "prom/pushgateway")
	v.SetDefault("cluster::monitoring::images::pushgateway::tag", "v1.0.0")

	v.SetDefault("cluster::logging::enabled", true)
	v.SetDefault("cluster::logging::namespace", "")
	v.SetDefault("cluster::logging::charts::operator::chart", "banzaicloud-stable/logging-operator")
	v.SetDefault("cluster::logging::charts::operator::version", "2.7.2")
	v.SetDefault("cluster::logging::charts::operator::values", map[string]interface{}{})
	v.SetDefault("cluster::logging::images::operator::repository", "banzaicloud/logging-operator")
	v.SetDefault("cluster::logging::images::operator::tag", "2.7.0")
	v.SetDefault("cluster::logging::charts::loki::chart", "banzaicloud-stable/loki")
	v.SetDefault("cluster::logging::charts::loki::version", "0.16.1")
	v.SetDefault("cluster::logging::charts::loki::values", map[string]interface{}{})
	v.SetDefault("cluster::logging::images::loki::repository", "grafana/loki")
	v.SetDefault("cluster::logging::images::loki::tag", "v0.3.0")
	v.SetDefault("cluster::logging::images::fluentbit::repository", "fluent/fluent-bit")
	v.SetDefault("cluster::logging::images::fluentbit::tag", "1.3.2")
	v.SetDefault("cluster::logging::images::fluentd::repository", "banzaicloud/fluentd")
	v.SetDefault("cluster::logging::images::fluentd::tag", "v1.7.4-alpine-10")

	v.SetDefault("cluster::dns::enabled", true)
	v.SetDefault("cluster::dns::namespace", "")
	v.SetDefault("cluster::dns::baseDomain", "")
	v.SetDefault("cluster::dns::providerSecret", "secret/data/banzaicloud/aws")
	v.SetDefault("cluster::dns::charts::externalDns::chart", "stable/external-dns")
	v.SetDefault("cluster::dns::charts::externalDns::version", "2.15.2")
	v.SetDefault("cluster::dns::charts::externalDns::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "bitnami/external-dns",
			"tag":        "0.5.18",
		},
	})

	v.SetDefault("cluster::autoscale::namespace", "")
	v.SetDefault("cluster::autoscale::hpa::prometheus::serviceName", "monitor-prometheus-server")
	v.SetDefault("cluster::autoscale::hpa::prometheus::serviceContext", "prometheus")
	v.SetDefault("cluster::autoscale::hpa::prometheus::localPort", 9090)
	v.SetDefault("cluster::autoscale::charts::clusterAutoscaler::chart", "stable/cluster-autoscaler")
	v.SetDefault("cluster::autoscale::charts::clusterAutoscaler::version", "6.2.0")
	v.SetDefault("cluster::autoscale::charts::clusterAutoscaler::values", map[string]interface{}{})
	v.SetDefault("cluster::autoscale::charts::clusterAutoscaler::imageVersionConstraints", []interface{}{
		map[string]interface{}{
			"k8sVersion": "<=1.12.x",
			"tag":        "v1.12.8",
			"repository": "gcr.io/google-containers/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": "~1.13",
			"tag":        "v1.13.9",
			"repository": "gcr.io/google-containers/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": "~1.14",
			"tag":        "v1.14.7",
			"repository": "gcr.io/google-containers/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": "~1.15",
			"tag":        "v1.15.4",
			"repository": "gcr.io/google-containers/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": "~1.16",
			"tag":        "v1.16.3",
			"repository": "gcr.io/google-containers/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": ">=1.17",
			"tag":        "v1.17.0",
			"repository": "gcr.io/google-containers/cluster-autoscaler",
		},
	})

	v.SetDefault("cluster::autoscale::charts::hpaOperator::chart", "banzaicloud-stable/hpa-operator")
	v.SetDefault("cluster::autoscale::charts::hpaOperator::version", "0.0.16")
	v.SetDefault("cluster::autoscale::charts::hpaOperator::values", map[string]interface{}{})

	v.SetDefault("cluster::securityScan::enabled", true)
	v.SetDefault("cluster::securityScan::anchore::enabled", false)
	v.SetDefault("cluster::securityScan::anchore::endpoint", "")
	v.SetDefault("cluster::securityScan::anchore::user", "")
	v.SetDefault("cluster::securityScan::anchore::password", "")

	v.SetDefault("cluster::expiry::enabled", true)

	v.SetDefault("cluster::disasterRecovery::enabled", true)
	v.SetDefault("cluster::disasterRecovery::namespace", "")
	v.SetDefault("cluster::disasterRecovery::ark::syncEnabled", true)
	v.SetDefault("cluster::disasterRecovery::ark::bucketSyncInterval", "10m")
	v.SetDefault("cluster::disasterRecovery::ark::restoreSyncInterval", "20s")
	v.SetDefault("cluster::disasterRecovery::ark::backupSyncInterval", "20s")
	v.SetDefault("cluster::disasterRecovery::ark::restoreWaitTimeout", "5m")
	v.SetDefault("cluster::disasterRecovery::charts::ark::chart", "banzaicloud-stable/ark")
	v.SetDefault("cluster::disasterRecovery::charts::ark::version", "1.2.2")
	v.SetDefault("cluster::disasterRecovery::charts::ark::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "banzaicloud/ark",
			"tag":        "v0.9.11",
			"pullPolicy": "IfNotPresent",
		},
	})

	v.SetDefault("cluster::backyards::enabled", true)
	v.SetDefault("cluster::backyards::istio::grafanaDashboardLocation", "./etc/dashboards/istio")
	v.SetDefault("cluster::backyards::istio::pilotImage", "banzaicloud/istio-pilot:1.3.4-bzc")
	v.SetDefault("cluster::backyards::istio::mixerImage", "banzaicloud/istio-mixer:1.3.4-bzc")
	v.SetDefault("cluster::backyards::charts::istioOperator::chart", "banzaicloud-stable/istio-operator")
	v.SetDefault("cluster::backyards::charts::istioOperator::version", "0.0.24")
	v.SetDefault("cluster::backyards::charts::istioOperator::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "banzaicloud/istio-operator",
			"tag":        "0.3.5",
		},
	})
	v.SetDefault("cluster::backyards::charts::backyards::chart", "banzaicloud-stable/backyards")
	v.SetDefault("cluster::backyards::charts::backyards::version", "1.0.4")
	v.SetDefault("cluster::backyards::charts::backyards::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "banzaicloud/backyards",
			"tag":        "1.0.4",
		},
	})
	v.SetDefault("cluster::backyards::charts::canaryOperator::chart", "banzaicloud-stable/canary-operator")
	v.SetDefault("cluster::backyards::charts::canaryOperator::version", "0.1.7")
	v.SetDefault("cluster::backyards::charts::canaryOperator::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "banzaicloud/canary-operator",
			"tag":        "0.1.5",
		},
	})

	v.SetDefault("cluster::federation::charts::kubefed::chart", "kubefed-charts/kubefed")
	v.SetDefault("cluster::federation::charts::kubefed::version", "0.1.0-rc6")
	v.SetDefault("cluster::federation::charts::kubefed::values", map[string]interface{}{
		"controllermanager": map[string]interface{}{
			"repository": "quay.io/kubernetes-multicluster",
			"tag":        "v0.1.0-rc6",
		},
	})

	// Helm configuration
	v.SetDefault("helm::tiller::version", "v2.14.2")
	v.SetDefault("helm::home", "./var/cache")
	v.SetDefault("helm::repositories::stable", "https://kubernetes-charts.storage.googleapis.com")
	v.SetDefault("helm::repositories::banzaicloud-stable", "https://kubernetes-charts.banzaicloud.com")
	v.SetDefault("helm::repositories::loki", "https://grafana.github.io/loki/charts")

	// Cloud configuration
	v.SetDefault("cloud::amazon::defaultRegion", "us-west-1")
	v.SetDefault("cloud::alibaba::defaultRegion", "eu-central-1")

	v.SetDefault("distribution::eks::templateLocation", "./templates/eks")
	v.SetDefault("distribution::eks::exposeAdminKubeconfig", true)

	v.SetDefault("cloudinfo::endpoint", "")
	v.SetDefault("hollowtrees::endpoint", "")
	v.SetDefault("hollowtrees::tokenSigningKey", "")

	// Temporary hook flags
	v.SetDefault("hooks::domainHookDisabled", false)

	// CICD config
	v.SetDefault("cicd::enabled", false)
	v.SetDefault("cicd::url", "http://localhost:8000")
	v.SetDefault("cicd::insecure", false)
	v.SetDefault("cicd::scm", "github")
	v.SetDefault("cicd::database::dialect", "mysql")
	v.SetDefault("cicd::database::host", "")
	v.SetDefault("cicd::database::port", 3306)
	v.SetDefault("cicd::database::tls", "")
	v.SetDefault("cicd::database::user", "")
	v.SetDefault("cicd::database::password", "")
	v.SetDefault("cicd::database::name", "cicd")
	v.SetDefault("cicd::database::params", map[string]string{
		"charset": "utf8mb4",
	})
	v.SetDefault("cicd::database::queryLog", false)

	// Auth provider (Gitlab/Github) settings
	v.SetDefault("github::token", "")
	v.SetDefault("gitlab::url", "https://gitlab.com/")
	v.SetDefault("gitlab::token", "")

	// Spotguide config
	v.SetDefault("spotguide::allowPrereleases", false)
	v.SetDefault("spotguide::allowPrivateRepos", false)
	v.SetDefault("spotguide::syncInterval", 5*time.Minute)
	v.SetDefault("spotguide::sharedLibraryGitHubOrganization", "spotguides")

	v.SetDefault("secret::tls::defaultValidity", "8760h") // 1 year
}
