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

	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/logging"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/monitoring"
)

// ClusterConfig contains cluster configuration.
type ClusterConfig struct {
	// Initial manifest
	Manifest string

	// Namespace to install Pipeline components to
	Namespace string

	// Features
	Vault        ClusterVaultConfig
	Monitoring   ClusterMonitoringConfig
	Logging      ClusterLoggingConfig
	SecurityScan ClusterSecurityScanConfig
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

	if c.SecurityScan.Enabled {
		if err := c.SecurityScan.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Process post-processes the configuration after loading (before validation).
func (c *ClusterConfig) Process() error {
	if c.Monitoring.Namespace == "" {
		c.Monitoring.Namespace = c.Namespace
	}

	if c.Logging.Namespace == "" {
		c.Logging.Namespace = c.Namespace
	}

	return nil
}

// ClusterVaultConfig contains cluster vault configuration.
type ClusterVaultConfig struct {
	Enabled bool
	Managed ClusterVaultManagedConfig
}

// ClusterVaultManagedConfig contains cluster vault configuration.
type ClusterVaultManagedConfig struct {
	Enabled bool
}

// ClusterMonitoringConfig contains cluster monitoring configuration.
type ClusterMonitoringConfig struct {
	Enabled bool

	monitoring.Config `mapstructure:",squash"`
}

// ClusterLoggingConfig contains cluster monitoring configuration.
type ClusterLoggingConfig struct {
	Enabled bool

	logging.Config `mapstructure:",squash"`
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

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, _ *pflag.FlagSet) {
	// Log configuration
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	v.SetDefault("log.format", "logfmt")
	v.SetDefault("log.level", "debug")
	v.RegisterAlias("log.noColor", "no_color")

	// ErrorHandler configuration
	v.SetDefault("errors.stackdriver.enabled", false)
	v.SetDefault("errors.stackdriver.projectId", false)

	// Cadence configuration
	v.SetDefault("cadence.host", "")
	v.SetDefault("cadence.port", 7933)
	v.SetDefault("cadence.domain", "pipeline")

	// Cluster configuration
	v.SetDefault("cluster.manifest", "")
	v.SetDefault("cluster.namespace", "pipeline-system")

	v.SetDefault("cluster.monitoring.enabled", true)
	v.SetDefault("cluster.monitoring.namespace", "")
	v.SetDefault("cluster.monitoring.grafana.adminUser", "admin")
	v.SetDefault("cluster.monitoring.charts.operator.chart", "stable/prometheus-operator")
	v.SetDefault("cluster.monitoring.charts.operator.version", "7.2.0")
	v.SetDefault("cluster.monitoring.charts.operator.values", map[string]interface{}{
		"prometheusOperator": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "quay.io/coreos/prometheus-operator",
				"tag":        "v0.32.0",
			},
		},
		"prometheus": map[string]interface{}{
			"prometheusSpec": map[string]interface{}{
				"image": map[string]interface{}{
					"repository": "quay.io/prometheus/prometheus",
					"tag":        "v2.12.0",
				},
			},
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"traefik.frontend.rule.type":                 "PathPrefix",
					"traefik.ingress.kubernetes.io/ssl-redirect": "true",
				},
			},
		},
		"alertmanager": map[string]interface{}{
			"alertmanagerSpec": map[string]interface{}{
				"image": map[string]interface{}{
					"repository": "quay.io/prometheus/alertmanager",
					"tag":        "v0.19.0",
				},
			},
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"traefik.frontend.rule.type":                 "PathPrefix",
					"traefik.ingress.kubernetes.io/ssl-redirect": "true",
				},
			},
		},
		"grafana": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "grafana/grafana",
				"tag":        "6.4.2",
			},
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"traefik.frontend.rule.type":                         "PathPrefixStrip",
					"traefik.ingress.kubernetes.io/redirect-permanent":   "true",
					"traefik.ingress.kubernetes.io/redirect-regex":       "^http://(.*)",
					"traefik.ingress.kubernetes.io/redirect-replacement": `https://$1\`,
				},
			},
		},
		"kube-state-metrics": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "quay.io/coreos/kube-state-metrics",
				"tag":        "v1.8.0",
			},
		},
		"prometheus-node-exporter": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "quay.io/prometheus/node-exporter",
				"tag":        "v0.18.0",
			},
		},
	})
	v.SetDefault("cluster.monitoring.charts.pushgateway.chart", "stable/prometheus-pushgateway")
	v.SetDefault("cluster.monitoring.charts.pushgateway.version", "1.0.1")
	v.SetDefault("cluster.monitoring.charts.pushgateway.values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "prom/pushgateway",
			"tag":        "v1.0.0",
		},
		"ingress": map[string]interface{}{
			"annotations": map[string]interface{}{
				"traefik.frontend.rule.type":                 "PathPrefix",
				"traefik.ingress.kubernetes.io/ssl-redirect": "true",
			},
		},
	})

	v.SetDefault("cluster.logging.enabled", true)
	v.SetDefault("cluster.logging.namespace", "")
	v.SetDefault("cluster.logging.charts.operator.chart", "banzaicloud-stable/logging-operator")
	v.SetDefault("cluster.logging.charts.operator.version", "0.3.3")
	v.SetDefault("cluster.logging.charts.operator.values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "banzaicloud/logging-operator",
			"tag":        "1.0.0",
		},
	})

	v.SetDefault("cluster.securityScan.anchore.enabled", false)
	v.SetDefault("cluster.securityScan.anchore.endpoint", "")
	v.SetDefault("cluster.securityScan.anchore.user", "")
	v.SetDefault("cluster.securityScan.anchore.password", "")

	// Helm configuration
	v.SetDefault("helm.tiller.version", "v2.14.2")
	v.SetDefault("helm.home", "./var/cache")
	v.SetDefault("helm.repositories.stable", "https://kubernetes-charts.storage.googleapis.com")
	v.SetDefault("helm.repositories.banzaicloud-stable", "https://kubernetes-charts.banzaicloud.com")
	v.SetDefault("helm.repositories.loki", "https://grafana.github.io/loki/charts")

	// Cloud configuration
	v.SetDefault("cloud.amazon.defaultRegion", "us-west-1")
	v.SetDefault("cloud.alibaba.defaultRegion", "eu-central-1")
}
