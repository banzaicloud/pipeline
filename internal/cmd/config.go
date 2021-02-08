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
	"os"
	"time"

	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterconfig"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/integratedservices/operator"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/logging"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/monitoring"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/vault"
	"github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/values"
)

type Config struct {
	// Cadence configuration
	Cadence cadence.Config

	Cloud CloudConfig

	Cloudinfo CloudinfoConfig

	// Cluster configuration
	Cluster ClusterConfig

	// Database configuration
	Database struct {
		database.Config `mapstructure:",squash"`

		AutoMigrate bool
	}

	Dex struct {
		APIAddr string
		APICa   string
	}

	Distribution struct {
		EKS struct {
			TemplateLocation            string
			DefaultNodeVolumeEncryption *struct {
				Enabled          bool
				EncryptionKeyARN string
			}
			DefaultNodeVolumeSize int
			ExposeAdminKubeconfig bool
			SSH                   struct {
				Generate bool
			}
		}

		PKE struct {
			Amazon struct {
				GlobalRegion           string
				DefaultImages          map[string]string
				DefaultNetworkProvider string
				DefaultNodeVolumeSize  int
			}
		}
	}

	Helm helm.Config

	Hollowtrees struct {
		Endpoint        string
		TokenSigningKey string
	}

	Kubernetes struct {
		Client struct {
			ForceGlobal bool
		}
	}

	// Log configuration
	Log log.Config

	Secret struct {
		TLS struct {
			DefaultValidity time.Duration
		}
	}

	// Telemetry configuration
	Telemetry TelemetryConfig

	// temporary switch to control the integrated service implementation
	IntegratedService struct {
		V2 bool
		// Integrated Service Operator related configuration
		Operator operator.Config
	}
}

func (c Config) Validate() error {
	var err error

	err = errors.Append(err, c.Cadence.Validate())

	err = errors.Append(err, c.Cloudinfo.Validate())

	err = errors.Append(err, c.Cluster.Validate())

	err = errors.Append(err, c.Database.Validate())

	err = errors.Append(err, c.Telemetry.Validate())

	err = errors.Append(err, c.Helm.Validate())

	return err
}

func (c Config) validateDistribution() error {
	pkeDefaultNP := c.Distribution.PKE.Amazon.DefaultNetworkProvider
	if pkeDefaultNP != "calico" && pkeDefaultNP != "cilium" {
		return errors.New("pke aws: default network provider must be calico or cilium")
	}

	return nil
}

func (c *Config) Process() error {
	var err error

	err = errors.Append(err, c.Cluster.Process())

	return err
}

type CloudConfig struct {
	Amazon AmazonCloudConfig
}

func (c CloudConfig) Validate() error {
	var errs error

	errs = errors.Append(errs, c.Amazon.Validate())

	return errs
}

type AmazonCloudConfig struct {
	DefaultRegion string
}

func (c AmazonCloudConfig) Validate() error {
	var errs error

	if c.DefaultRegion == "" {
		errs = errors.Append(errs, errors.New("amazon default region is required"))
	}

	return errs
}

type CloudinfoConfig struct {
	Endpoint string
}

func (c CloudinfoConfig) Validate() error {
	var err error

	if c.Endpoint == "" {
		err = errors.Append(err, errors.New("cloudinfo endpoint is required"))
	}

	return err
}

// ClusterConfig contains cluster configuration.
type ClusterConfig struct {
	Autoscale ClusterAutoscaleConfig

	Backyards istiofeature.StaticConfig

	DisasterRecovery ClusterDisasterRecoveryConfig

	DNS ClusterDNSConfig

	Expiry ClusterExpiryConfig

	Federation federation.StaticConfig

	Ingress ClusterIngressConfig

	Labels clusterconfig.LabelConfig

	// Initial manifest
	Manifest string

	Monitoring ClusterMonitoringConfig

	Logging ClusterLoggingConfig

	// Namespace to install Pipeline components to
	Namespace string

	// Posthook configs
	PostHook cluster.PostHookConfig

	SecurityScan ClusterSecurityScanConfig

	Vault ClusterVaultConfig
}

// Validate validates the configuration.
func (c ClusterConfig) Validate() error {
	var errs error

	errs = errors.Append(errs, c.DNS.Validate())

	errs = errors.Append(errs, c.Ingress.Validate())

	errs = errors.Append(errs, c.Labels.Validate())

	errs = errors.Append(errs, c.Logging.Validate())

	if c.Manifest != "" {
		file, err := os.OpenFile(c.Manifest, os.O_RDONLY, 0666)
		_ = file.Close()
		if err != nil {
			errs = errors.Append(errs, errors.Wrap(err, "cluster manifest file is not readable"))
		}
	}

	errs = errors.Append(errs, c.Monitoring.Validate())

	if c.Namespace == "" {
		errs = errors.Append(errs, errors.New("cluster namespace is required"))
	}

	errs = errors.Append(errs, c.SecurityScan.Validate())

	errs = errors.Append(errs, c.Vault.Validate())

	return errs
}

// Process post-processes the configuration after loading (before validation).
func (c *ClusterConfig) Process() error {
	if c.Autoscale.Namespace == "" {
		c.Autoscale.Namespace = c.Namespace
	}

	if c.DisasterRecovery.Namespace == "" {
		c.DisasterRecovery.Namespace = c.Namespace
	}

	if c.DNS.Namespace == "" {
		c.DNS.Namespace = c.Namespace
	}

	if c.Ingress.Namespace == "" {
		c.Ingress.Namespace = c.Namespace
	}

	if c.Labels.Namespace == "" {
		c.Labels.Namespace = c.Namespace
	}

	if c.Logging.Namespace == "" {
		c.Logging.Namespace = c.Namespace
	}

	if c.Monitoring.Namespace == "" {
		c.Monitoring.Namespace = c.Namespace
	}

	if c.SecurityScan.PipelineNamespace == "" {
		c.SecurityScan.PipelineNamespace = c.Namespace
	}

	if c.Vault.Namespace == "" {
		c.Vault.Namespace = c.Namespace
	}

	return nil
}

type ClusterAutoscaleConfig struct {
	Namespace string

	HPA struct {
		Prometheus struct {
			ServiceName    string
			ServiceContext string
			LocalPort      int
		}
	}

	Charts struct {
		ClusterAutoscaler struct {
			Chart                   string
			Version                 string
			ImageVersionConstraints []struct {
				K8sVersion string
				Tag        string
				Repository string
			}
		}

		HPAOperator struct {
			Chart   string
			Version string
			Values  values.Config
		}
	}
}

type ClusterDisasterRecoveryConfig struct {
	Namespace string

	Ark struct {
		SyncEnabled         bool
		BucketSyncInterval  time.Duration
		RestoreSyncInterval time.Duration
		BackupSyncInterval  time.Duration
		RestoreWaitTimeout  time.Duration
	}

	Charts struct {
		Ark struct {
			Chart   string
			Version string
			Values  struct {
				Image struct {
					Repository string
					Tag        string
					PullPolicy string
				}
				AwsPluginImage struct {
					Repository string
					Tag        string
					PullPolicy string
				}
				AzurePluginImage struct {
					Repository string
					Tag        string
					PullPolicy string
				}
				GcpPluginImage struct {
					Repository string
					Tag        string
					PullPolicy string
				}
			}
		}
	}
}

// ClusterDNSConfig contains cluster DNS configuration.
type ClusterDNSConfig struct {
	Enabled bool

	dns.Config `mapstructure:",squash"`
}

func (c ClusterDNSConfig) Validate() error {
	var errs error

	if c.Enabled {
		errs = errors.Append(errs, c.Config.Validate())
	}

	return errs
}

type ClusterExpiryConfig struct {
	Enabled bool
}

type ClusterIngressConfig struct {
	Enabled bool

	ingress.Config `mapstructure:",squash"`
}

func (c ClusterIngressConfig) Validate() error {
	var errs error

	if c.Enabled {
		errs = errors.Append(errs, c.Config.Validate())
	}

	return errs
}

// ClusterLoggingConfig contains cluster logging configuration.
type ClusterLoggingConfig struct {
	Enabled bool

	logging.Config `mapstructure:",squash"`
}

func (c ClusterLoggingConfig) Validate() error {
	var errs error

	if c.Enabled {
		errs = errors.Append(errs, c.Config.Validate())
	}

	return errs
}

// ClusterMonitoringConfig contains cluster monitoring configuration.
type ClusterMonitoringConfig struct {
	Enabled bool

	monitoring.Config `mapstructure:",squash"`
}

func (c ClusterMonitoringConfig) Validate() error {
	var errs error

	if c.Enabled {
		errs = errors.Append(errs, c.Config.Validate())
	}

	return errs
}

// ClusterSecurityScanConfig contains cluster security scan configuration.
type ClusterSecurityScanConfig struct {
	Enabled bool

	securityscan.Config `mapstructure:",squash"`
}

func (c ClusterSecurityScanConfig) Validate() error {
	var err error

	if c.Enabled {
		err = errors.Append(err, c.Config.Validate())
	}

	return err
}

// ClusterVaultConfig contains cluster vault configuration.
type ClusterVaultConfig struct {
	Enabled bool

	vault.Config `mapstructure:",squash"`
}

func (c ClusterVaultConfig) Validate() error {
	var errs error

	if c.Enabled {
		errs = errors.Append(errs, c.Config.Validate())
	}

	return errs
}

// TelemetryConfig contains telemetry configuration.
type TelemetryConfig struct {
	Enabled bool
	Addr    string
	Debug   bool
}

// Validate validates the configuration.
func (c TelemetryConfig) Validate() error {
	var err error

	if c.Enabled {
		if c.Addr == "" {
			err = errors.Append(err, errors.New("telemetry http server address is required"))
		}
	}

	return err
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, p *pflag.FlagSet) {
	// Log configuration
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	v.SetDefault("auth::token::signingKey", "")
	v.SetDefault("auth::token::issuer", "")
	v.SetDefault("auth::token::audience", "")

	v.SetDefault("log::format", "json")
	v.SetDefault("log::level", "info")
	v.RegisterAlias("log::noColor", "no_color")

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
	v.SetDefault("cluster::labels::charts::nodepoolLabelOperator::version", "0.1.1")
	v.SetDefault("cluster::labels::charts::nodepoolLabelOperator::values", map[string]interface{}{})

	v.SetDefault("cluster::vault::enabled", true)
	v.SetDefault("cluster::vault::namespace", "")
	v.SetDefault("cluster::vault::managed::enabled", false)
	v.SetDefault("cluster::vault::managed::endpoint", "")
	v.SetDefault("cluster::vault::charts::webhook::chart", "banzaicloud-stable/vault-secrets-webhook")
	v.SetDefault("cluster::vault::charts::webhook::version", "1.10.1")
	v.SetDefault("cluster::vault::charts::webhook::values", map[string]interface{}{})

	v.SetDefault("cluster::monitoring::enabled", true)
	v.SetDefault("cluster::monitoring::namespace", "")
	v.SetDefault("cluster::monitoring::grafana::adminUser", "admin")
	v.SetDefault("cluster::monitoring::charts::operator::chart", "prometheus-community/kube-prometheus-stack")
	v.SetDefault("cluster::monitoring::charts::operator::version", "12.11.3")
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
			"sidecar": map[string]interface{}{
				"datasources": map[string]interface{}{
					"enabled": "true",
				},
			},
		},
	})
	v.SetDefault("cluster::monitoring::images::operator::repository", "quay.io/prometheus-operator/prometheus-operator")
	v.SetDefault("cluster::monitoring::images::operator::tag", "v0.44.1")
	v.SetDefault("cluster::monitoring::images::prometheus::repository", "quay.io/prometheus/prometheus")
	v.SetDefault("cluster::monitoring::images::prometheus::tag", "v2.22.1")
	v.SetDefault("cluster::monitoring::images::alertmanager::repository", "quay.io/prometheus/alertmanager")
	v.SetDefault("cluster::monitoring::images::alertmanager::tag", "v0.21.0")
	v.SetDefault("cluster::monitoring::images::grafana::repository", "grafana/grafana")
	v.SetDefault("cluster::monitoring::images::grafana::tag", "7.3.5")
	v.SetDefault("cluster::monitoring::images::kubestatemetrics::repository", "quay.io/coreos/kube-state-metrics")
	v.SetDefault("cluster::monitoring::images::kubestatemetrics::tag", "v1.9.7")
	v.SetDefault("cluster::monitoring::images::nodeexporter::repository", "quay.io/prometheus/node-exporter")
	v.SetDefault("cluster::monitoring::images::nodeexporter::tag", "v1.0.1")

	v.SetDefault("cluster::monitoring::charts::pushgateway::chart", "prometheus-community/prometheus-pushgateway")
	v.SetDefault("cluster::monitoring::charts::pushgateway::version", "1.5.1")
	v.SetDefault("cluster::monitoring::charts::pushgateway::values", map[string]interface{}{})
	v.SetDefault("cluster::monitoring::images::pushgateway::repository", "prom/pushgateway")
	v.SetDefault("cluster::monitoring::images::pushgateway::tag", "v1.0.1")

	v.SetDefault("cluster::logging::enabled", true)
	v.SetDefault("cluster::logging::namespace", "")
	v.SetDefault("cluster::logging::charts::operator::chart", "banzaicloud-stable/logging-operator")
	v.SetDefault("cluster::logging::charts::operator::version", "3.2.2")
	v.SetDefault("cluster::logging::charts::operator::values", map[string]interface{}{})
	v.SetDefault("cluster::logging::images::operator::repository", "banzaicloud/logging-operator")
	v.SetDefault("cluster::logging::images::operator::tag", "3.2.0")
	v.SetDefault("cluster::logging::charts::loki::chart", "banzaicloud-stable/loki")
	v.SetDefault("cluster::logging::charts::loki::version", "0.17.4")
	v.SetDefault("cluster::logging::charts::loki::values", map[string]interface{}{})
	v.SetDefault("cluster::logging::images::loki::repository", "grafana/loki")
	v.SetDefault("cluster::logging::images::loki::tag", "v1.3.0")
	v.SetDefault("cluster::logging::images::fluentbit::repository", "fluent/fluent-bit")
	v.SetDefault("cluster::logging::images::fluentbit::tag", "1.4.4")
	v.SetDefault("cluster::logging::images::fluentd::repository", "banzaicloud/fluentd")
	v.SetDefault("cluster::logging::images::fluentd::tag", "v1.10.3-alpine-2")

	v.SetDefault("cluster::dns::enabled", true)
	v.SetDefault("cluster::dns::namespace", "")
	v.SetDefault("cluster::dns::baseDomain", "")
	v.SetDefault("cluster::dns::providerSecret", "secret/data/banzaicloud/aws")
	v.SetDefault("cluster::dns::charts::externalDns::chart", "bitnami/external-dns")
	v.SetDefault("cluster::dns::charts::externalDns::version", "4.5.0")
	v.SetDefault("cluster::dns::charts::externalDns::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "k8s.gcr.io/external-dns/external-dns",
			"tag":        "v0.7.5",
		},
	})

	v.SetDefault("cluster::ingress::enabled", false)
	v.SetDefault("cluster::ingress::controllers", []string{"traefik"})
	v.SetDefault("cluster::ingress::namespace", "")
	v.SetDefault("cluster::ingress::releaseName", "ingress")
	v.SetDefault("cluster::ingress::charts::traefik::chart", "stable/traefik")
	v.SetDefault("cluster::ingress::charts::traefik::version", "1.86.2")
	v.SetDefault("cluster::ingress::charts::traefik::values", `
ssl:
  enabled: true
  generateTLS: true
rbac:
  enabled: true
`)

	v.SetDefault("cluster::autoscale::namespace", "")
	v.SetDefault("cluster::autoscale::hpa::prometheus::serviceName", "monitor-prometheus-operato-prometheus")
	v.SetDefault("cluster::autoscale::hpa::prometheus::serviceContext", "prometheus")
	v.SetDefault("cluster::autoscale::hpa::prometheus::localPort", 9090)
	v.SetDefault("cluster::autoscale::charts::clusterAutoscaler::chart", "stable/cluster-autoscaler")
	v.SetDefault("cluster::autoscale::charts::clusterAutoscaler::version", "7.1.0")
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
			"tag":        "v1.16.7",
			"repository": "k8s.gcr.io/autoscaling/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": "~1.17",
			"tag":        "v1.17.4",
			"repository": "k8s.gcr.io/autoscaling/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": "~1.18",
			"tag":        "v1.18.3",
			"repository": "k8s.gcr.io/autoscaling/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": ">=1.19",
			"tag":        "v1.19.1",
			"repository": "k8s.gcr.io/autoscaling/cluster-autoscaler",
		},
		map[string]interface{}{
			"k8sVersion": ">=1.20",
			"tag":        "v1.20.0",
			"repository": "k8s.gcr.io/autoscaling/cluster-autoscaler",
		},
	})

	v.SetDefault("cluster::autoscale::charts::hpaOperator::chart", "banzaicloud-stable/hpa-operator")
	v.SetDefault("cluster::autoscale::charts::hpaOperator::version", "0.3.0")
	v.SetDefault("cluster::autoscale::charts::hpaOperator::values", map[string]interface{}{})

	v.SetDefault("cluster::securityScan::enabled", true)
	v.SetDefault("cluster::securityScan::anchore::enabled", false)
	v.SetDefault("cluster::securityScan::anchore::endpoint", "")
	v.SetDefault("cluster::securityScan::anchore::user", "")
	v.SetDefault("cluster::securityScan::anchore::password", "")
	v.SetDefault("cluster::securityScan::anchore::insecure", false)
	v.SetDefault("cluster::securityScan::webhook::chart", "banzaicloud-stable/anchore-policy-validator")
	v.SetDefault("cluster::securityScan::webhook::version", "0.6.2")
	v.SetDefault("cluster::securityScan::webhook::release", "anchore")
	v.SetDefault("cluster::securityScan::webhook::namespace", "pipeline-system")
	// v.SetDefault("cluster::securityScan::webhook::values", map[string]interface{}{
	//	"image": map[string]interface{}{
	//		"repository": "banzaicloud/ark",
	//		"tag":        "v0.9.11",
	//		"pullPolicy": "IfNotPresent",
	//	},
	// })
	v.SetDefault("cluster::securityScan::anchore::policyPath", "/policies")

	v.SetDefault("cluster::expiry::enabled", true)

	// ingress controller config
	v.SetDefault("cluster::posthook::ingress::enabled", true)
	v.SetDefault("cluster::posthook::ingress::chart", "banzaicloud-stable/pipeline-cluster-ingress")
	v.SetDefault("cluster::posthook::ingress::version", "0.0.10")
	v.SetDefault("cluster::posthook::ingress::values", `
traefik:
  ssl:
    enabled: true
    generateTLS: true
`)

	// Kubernetes Dashboard
	v.SetDefault("cluster::posthook::dashboard::enabled", false)
	v.SetDefault("cluster::posthook::dashboard::chart", "banzaicloud-stable/kubernetes-dashboard")
	v.SetDefault("cluster::posthook::dashboard::version", "0.9.2")

	// Init spot config
	v.SetDefault("cluster::posthook::spotconfig::enabled", false)
	v.SetDefault("cluster::posthook::spotconfig::charts::scheduler::chart", "banzaicloud-stable/spot-scheduler")
	v.SetDefault("cluster::posthook::spotconfig::charts::scheduler::version", "0.1.2")
	v.SetDefault("cluster::posthook::spotconfig::charts::webhook::chart", "banzaicloud-stable/spot-config-webhook")
	v.SetDefault("cluster::posthook::spotconfig::charts::webhook::version", "0.1.6")

	// Instance Termination Handler
	v.SetDefault("cluster::posthook::ith::enabled", true)
	v.SetDefault("cluster::posthook::ith::chart", "banzaicloud-stable/instance-termination-handler")
	v.SetDefault("cluster::posthook::ith::version", "0.0.9")

	// Horizontal Pod Autoscaler
	v.SetDefault("cluster::posthook::hpa::enabled", false)

	// Cluster Autoscaler
	v.SetDefault("cluster::posthook::autoscaler::enabled", true)

	// v.SetDefault("cluster::disasterRecovery::enabled", true)
	v.SetDefault("cluster::disasterRecovery::namespace", "pipeline-system")
	v.SetDefault("cluster::disasterRecovery::ark::syncEnabled", true)
	v.SetDefault("cluster::disasterRecovery::ark::bucketSyncInterval", "10m")
	v.SetDefault("cluster::disasterRecovery::ark::restoreSyncInterval", "20s")
	v.SetDefault("cluster::disasterRecovery::ark::backupSyncInterval", "20s")
	v.SetDefault("cluster::disasterRecovery::ark::restoreWaitTimeout", "5m")
	v.SetDefault("cluster::disasterRecovery::charts::ark::chart", "banzaicloud-stable/velero")
	v.SetDefault("cluster::disasterRecovery::charts::ark::version", "2.13.3-bc.1")
	v.SetDefault("cluster::disasterRecovery::charts::ark::values", map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "velero/velero",
			"tag":        "v1.5.1",
			"pullPolicy": "IfNotPresent",
		},
		"awsPluginImage": map[string]interface{}{
			"repository": "velero/velero-plugin-for-aws",
			"tag":        "v1.1.0",
			"pullPolicy": "IfNotPresent",
		},
		"azurePluginImage": map[string]interface{}{
			"repository": "velero/velero-plugin-for-microsoft-azure",
			"tag":        "v1.1.0",
			"pullPolicy": "IfNotPresent",
		},
		"gcpPluginImage": map[string]interface{}{
			"repository": "velero/velero-plugin-for-gcp",
			"tag":        "v1.1.0",
			"pullPolicy": "IfNotPresent",
		},
	})

	// v.SetDefault("cluster::backyards::enabled", true)
	v.SetDefault("cluster::backyards::istio::grafanaDashboardLocation", "./etc/dashboards/istio")
	v.SetDefault("cluster::backyards::istio::pilotImage", "banzaicloud/istio-pilot:1.5.1-bzc.3")
	v.SetDefault("cluster::backyards::istio::mixerImage", "banzaicloud/istio-mixer:1.5.1-bzc.3")
	v.SetDefault("cluster::backyards::istio::proxyImage", "banzaicloud/istio-proxyv2:1.5.1-bzc.3")
	v.SetDefault("cluster::backyards::istio::sidecarInjectorImage", "banzaicloud/istio-sidecar-injector:1.5.1-bzc.3")
	v.SetDefault("cluster::backyards::charts::istioOperator::chart", "banzaicloud-stable/istio-operator")
	v.SetDefault("cluster::backyards::charts::istioOperator::version", "0.0.45")
	v.SetDefault("cluster::backyards::charts::istioOperator::values", map[string]interface{}{
		"operator": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "banzaicloud/istio-operator",
				"tag":        "0.5.5",
			},
		},
	})
	v.SetDefault("cluster::backyards::charts::backyards::chart", "banzaicloud-stable/backyards")
	v.SetDefault("cluster::backyards::charts::backyards::version", "1.2.12")
	v.SetDefault("cluster::backyards::charts::backyards::values", map[string]interface{}{
		"application": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "banzaicloud/backyards",
				"tag":        "1.2.3",
			},
		},
		"web": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "banzaicloud/backyards-web",
				"tag":        "1.2.3",
			},
		},
	})
	v.SetDefault("cluster::backyards::charts::canaryOperator::chart", "banzaicloud-stable/canary-operator")
	v.SetDefault("cluster::backyards::charts::canaryOperator::version", "0.1.11")
	v.SetDefault("cluster::backyards::charts::canaryOperator::values", map[string]interface{}{
		"operator": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "banzaicloud/canary-operator",
				"tag":        "0.1.9",
			},
		},
	})
	v.SetDefault("cluster::backyards::charts::nodeexporter::chart", "stable/prometheus-node-exporter")
	v.SetDefault("cluster::backyards::charts::nodeexporter::version", "1.8.1")

	v.SetDefault("cluster::federation::charts::kubefed::chart", "kubefed-charts/kubefed")
	v.SetDefault("cluster::federation::charts::kubefed::version", "v0.4.0")
	v.SetDefault("cluster::federation::charts::kubefed::values", map[string]interface{}{
		"controllermanager": map[string]interface{}{
			"controller": map[string]interface{}{
				"tag": "v0.4.0",
			},
			"webhook": map[string]interface{}{
				"tag": "v0.4.0",
			},
		},
	})

	// Helm configuration
	v.SetDefault("helm::home", "./var/cache")
	v.SetDefault("helm::repositories::stable", "https://charts.helm.sh/stable")
	v.SetDefault("helm::repositories::banzaicloud-stable", "https://kubernetes-charts.banzaicloud.com")
	v.SetDefault("helm::repositories::bitnami", "https://charts.bitnami.com/bitnami")
	v.SetDefault("helm::repositories::loki", "https://grafana.github.io/loki/charts")
	v.SetDefault("helm::repositories::kubefed-charts", "https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts")
	v.SetDefault("helm::repositories::prometheus-community", "https://prometheus-community.github.io/helm-charts")

	// Cloud configuration
	v.SetDefault("cloud::amazon::defaultRegion", "us-west-1")

	v.SetDefault("distribution::eks::templateLocation", "./templates/eks")
	v.SetDefault("distribution::eks::exposeAdminKubeconfig", true)
	v.SetDefault("distribution::eks::ssh::generate", true)

	v.SetDefault("distribution::pke::amazon::globalRegion", "us-east-1")
	v.SetDefault("distribution::pke::amazon::defaultImages", map[string]string{})
	v.SetDefault("distribution::pke::amazon::defaultNetworkProvider", "cilium")

	v.SetDefault("cloudinfo::endpoint", "")
	v.SetDefault("hollowtrees::endpoint", "")
	v.SetDefault("hollowtrees::tokenSigningKey", "")

	v.SetDefault("secret::tls::defaultValidity", "8760h") // 1 year

	// Telemetry configuration
	v.SetDefault("telemetry::enabled", false)
	p.String("telemetry-addr", "127.0.0.1:9900", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry::addr", p.Lookup("telemetry-addr"))
	v.SetDefault("telemetry::addr", "127.0.0.1:9900")
	v.SetDefault("telemetry::debug", true)

	v.SetDefault("integratedservice::v2", false)

	// Integrated Service Operator
	v.SetDefault("integratedservice::operator::chart", "banzaicloud-stable/integrated-service-operator")
	v.SetDefault("integratedservice::operator::version", "0.4.0")
	v.SetDefault("integratedservice::operator::release", "iso")
	v.SetDefault("integratedservice::operator::namespace", "pipeline-system")
	v.SetDefault("integratedservice::operator::batchsize", 100)
}
