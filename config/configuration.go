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

package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/spf13/viper"
)

const (
	// local helm path
	helmPath = "helm.path"

	// DNSBaseDomain configuration key for the base domain setting
	DNSBaseDomain = "dns.domain"

	// DNSGcIntervalMinute configuration key for the interval setting at which the DNS garbage collector runs
	DNSGcIntervalMinute = "dns.gcIntervalMinute"

	// DNSGcLogLevel configuration key for the DNS garbage collector logging level default value: "debug"
	DNSGcLogLevel = "dns.gcLogLevel"

	// DNSExternalDnsChartVersion set the external-dns chart version default value: "1.6.2"
	DNSExternalDnsChartVersion = "dns.externalDnsChartVersion"

	// DNSExternalDnsImageVersion set the external-dns image version
	DNSExternalDnsImageVersion = "dns.externalDnsImageVersion"

	// Route53MaintenanceWndMinute configuration key for the maintenance window for Route53.
	// This is the maintenance window before the next AWS Route53 pricing period starts
	Route53MaintenanceWndMinute = "route53.maintenanceWindowMinute"

	//PipelineSystemNamespace pipeline infra namespace key
	PipelineSystemNamespace = "infra.namespace"

	//PipelineHeadNodePoolName name of our Head node pool for Pipeline Infra deployments
	PipelineHeadNodePoolName = "infra.headNodePoolName"

	HeadNodeTaintRetryAttempt      = "infra.headNodeTaintRetryAttempt"
	HeadNodeTaintRetrySleepSeconds = "infra.headNodeTaintRetrySleepSeconds"

	//PipelineLabelDomain reserved node pool label domains
	PipelineLabelDomain = "infra.pipelineLabelDomain"

	//ForbiddenLabelDomains reserved node pool label domains
	ForbiddenLabelDomains = "infra.forbiddenLabelDomains"

	// EksTemplateLocation is the configuration key the location to get EKS Cloud Formation templates from
	// the location to get EKS Cloud Formation templates from
	EksTemplateLocation = "eks.templateLocation"
	// EksASGFulfillmentTimeout configuration key for the timeout of EKS ASG instance fulfillments
	EksASGFulfillmentTimeout = "eks.ASGFulfillmentTimeout"

	// AwsCredentialPath is the path in Vault to get AWS credentials from for Pipeline
	AwsCredentialPath = "aws.credentials.path"

	// Config keys to GKE resource delete
	GKEResourceDeleteWaitAttempt  = "gke.resourceDeleteWaitAttempt"
	GKEResourceDeleteSleepSeconds = "gke.resourceDeleteSleepSeconds"

	// Config keys to OKE nodepool wait
	OKEWaitAttemptsForNodepoolActive = "oke.waitAttemptsForNodepoolActive"
	OKESleepSecondsForNodepoolActive = "oke.sleepSecondsForNodepoolActive"

	// Logging
	LoggingLogLevel  = "logging.loglevel"
	LoggingLogFormat = "logging.logformat"

	// ARK
	ARKName                = "ark.name"
	ARKNamespace           = "ark.namespace"
	ARKChart               = "ark.chart"
	ARKChartVersion        = "ark.chartVersion"
	ARKImage               = "ark.image"
	ARKImageTag            = "ark.imageTag"
	ARKPullPolicy          = "ark.pullPolicy"
	ARKSyncEnabled         = "ark.syncEnabled"
	ARKLogLevel            = "ark.logLevel"
	ARKBucketSyncInterval  = "ark.bucketSyncInterval"
	ARKRestoreSyncInterval = "ark.restoreSyncInterval"
	ARKBackupSyncInterval  = "ark.backupSyncInterval"
	ARKRestoreWaitTimeout  = "ark.restoreWaitTimeout"

	// Spot Metrics
	SpotMetricsEnabled            = "spotmetrics.enabled"
	SpotMetricsCollectionInterval = "spotmetrics.collectionInterval"

	// Database
	DBAutoMigrateEnabled = "database.autoMigrateEnabled"

	// Monitor config path
	MonitorEnabled                = "monitor.enabled"
	MonitorConfigMap              = "monitor.configMap"              // Prometheus config map
	MonitorConfigMapPrometheusKey = "monitor.configMapPrometheusKey" // Prometheus config key in the prometheus config map
	MonitorCertSecret             = "monitor.certSecret"             // Kubernetes secret for kubernetes cluster certs
	MonitorCertMountPath          = "monitor.mountPath"              // Mount path for the kubernetes cert secret
	// Monitor constants
	MonitorReleaseName = "monitor"

	// Metrics
	MetricsEnabled = "metrics.enabled"
	MetricsPort    = "metrics.port"
	MetricsAddress = "metrics.address"
	MetricsDebug   = "metrics.debug"

	ControlPlaneNamespace = "infra.control-plane-namespace" // Namespace where the pipeline and prometheus runs

	SetCookieDomain = "auth.setCookieDomain"

	// Logging operator constants
	LoggingReleaseName          = "logging-operator"
	LoggingOperatorChartVersion = "loggingOperator.chartVersion"
	LoggingOperatorImageTag     = "loggingOperator.imageTag"

	// Spotguides constants
	SpotguideAllowPrereleases                = "spotguide.allowPrereleases"
	SpotguideAllowPrivateRepos               = "spotguide.allowPrivateRepos"
	SpotguideSyncInterval                    = "spotguide.syncInterval"
	SpotguideSharedLibraryGitHubOrganization = "spotguide.sharedLibraryGitHubOrganization"

	// full endpoint url of CloudInfo for ex: https://alpha.dev.banzaicloud.com/cloudinfo/api/v1
	CloudInfoEndPoint = "cloudinfo.endpointUrl"

	IstioOperatorChartVersion     = "servicemesh.istioOperatorChartVersion"
	IstioGrafanaDashboardLocation = "servicemesh.grafanaDashboardLocation"

	// NodePool LabelSet Operator
	NodePoolLabelSetOperatorChartVersion = "nodepools.labelSetOperatorChartVersion"

	// Prometheus svc name, context & local port of Prometheus deploy if monitoring is enabled on cluster
	PrometheusServiceName    = "prometheus.serviceName"
	PrometheusServiceContext = "prometheus.serviceContext"
	PrometheusLocalPort      = "prometheus.localPort"

	// Default regions config keys to initialize clients
	AmazonInitializeRegionKey  = "amazon.defaultApiRegion"
	AlibabaInitializeRegionKey = "alibaba.defaultApiRegion"
)

//Init initializes the configurations
func init() {

	viper.AddConfigPath("$HOME/config")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$PIPELINE_CONFIG_DIR/")
	viper.SetConfigName("config")

	// Set defaults TODO expand defaults
	viper.SetDefault("cicd.url", "http://localhost:8000")
	viper.SetDefault("cicd.scm", "github")
	viper.SetDefault("helm.retryAttempt", 30)
	viper.SetDefault("helm.retrySleepSeconds", 15)
	viper.SetDefault("helm.tillerVersion", "v2.10.0")
	viper.SetDefault("helm.stableRepositoryURL", "https://kubernetes-charts.storage.googleapis.com")
	viper.SetDefault("helm.banzaiRepositoryURL", "http://kubernetes-charts.banzaicloud.com")
	viper.SetDefault(helmPath, "./orgs")
	viper.SetDefault("cloud.defaultProfileName", "default")
	viper.SetDefault("cloud.configRetryCount", 30)
	viper.SetDefault("cloud.configRetrySleep", 15)
	viper.SetDefault(AwsCredentialPath, "secret/data/banzaicloud/aws")
	viper.SetDefault(LoggingLogLevel, "debug")
	viper.SetDefault(LoggingLogFormat, "text")

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err.Error())
	}
	viper.SetDefault("statestore.path", fmt.Sprintf("%s/statestore/", pwd))

	viper.SetDefault("auth.jwtissuer", "https://banzaicloud.com/")
	viper.SetDefault("auth.jwtaudience", "https://pipeline.banzaicloud.com")
	viper.SetDefault("auth.secureCookie", true)
	viper.SetDefault("auth.dexURL", "http://127.0.0.1:5556/dex")
	viper.SetDefault("auth.dexGrpcAddress", "127.0.0.1:5557")
	viper.SetDefault("auth.dexGrpcCaCert", "")
	viper.SetDefault(SetCookieDomain, false)

	viper.SetDefault("pipeline.bindaddr", "127.0.0.1:9090")
	viper.SetDefault("pipeline.certfile", "")
	viper.SetDefault("pipeline.keyfile", "")
	viper.SetDefault("pipeline.uipath", "/ui")
	viper.SetDefault("pipeline.basepath", "")
	viper.SetDefault("pipeline.signupRedirectPath", "/ui")
	viper.SetDefault(MetricsEnabled, false)
	viper.SetDefault(MetricsPort, "9900")
	viper.SetDefault(MetricsAddress, "127.0.0.1")
	viper.SetDefault(MetricsDebug, true)
	viper.SetDefault("database.dialect", "mysql")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.user", "kellyslater")
	viper.SetDefault("database.password", "pipemaster123!")
	viper.SetDefault("database.dbname", "pipelinedb")
	viper.SetDefault("database.logging", false)
	viper.SetDefault(DBAutoMigrateEnabled, false)
	viper.SetDefault("audit.enabled", true)
	viper.SetDefault("audit.headers", []string{"secretId"})
	viper.SetDefault("audit.skippaths", []string{"/auth/github/callback", "/pipeline/api"})
	viper.SetDefault("tls.validity", "8760h") // 1 year
	viper.SetDefault(DNSBaseDomain, "example.org")
	viper.SetDefault(DNSGcIntervalMinute, 1)
	viper.SetDefault(DNSExternalDnsChartVersion, "1.6.2")
	viper.SetDefault(DNSExternalDnsImageVersion, "v0.5.11")
	viper.SetDefault(DNSGcLogLevel, "debug")
	viper.SetDefault(Route53MaintenanceWndMinute, 15)

	viper.SetDefault(GKEResourceDeleteWaitAttempt, 12)
	viper.SetDefault(GKEResourceDeleteSleepSeconds, 5)

	viper.SetDefault(OKEWaitAttemptsForNodepoolActive, 60)
	viper.SetDefault(OKESleepSecondsForNodepoolActive, 30)

	viper.SetDefault(ARKName, "ark")
	viper.SetDefault(ARKNamespace, "pipeline-infra")
	viper.SetDefault(ARKChart, "banzaicloud-stable/ark")
	viper.SetDefault(ARKChartVersion, "1.2.1")
	viper.SetDefault(ARKImage, "banzaicloud/ark")
	viper.SetDefault(ARKImageTag, "v0.9.6")
	viper.SetDefault(ARKPullPolicy, "IfNotPresent")
	viper.SetDefault(ARKSyncEnabled, true)
	viper.SetDefault(ARKLogLevel, "info")
	viper.SetDefault(ARKBucketSyncInterval, "10m")
	viper.SetDefault(ARKRestoreSyncInterval, "20s")
	viper.SetDefault(ARKBackupSyncInterval, "20s")
	viper.SetDefault(ARKRestoreWaitTimeout, "5m")

	viper.SetDefault(SpotMetricsEnabled, false)
	viper.SetDefault(SpotMetricsCollectionInterval, "30s")

	viper.SetDefault(MonitorEnabled, false)
	viper.SetDefault(MonitorConfigMap, "")
	viper.SetDefault(MonitorConfigMapPrometheusKey, "prometheus.yml")
	viper.SetDefault(MonitorCertSecret, "")
	viper.SetDefault(MonitorCertMountPath, "")
	viper.SetDefault("monitor.grafanaAdminUsername", "admin")

	// empty string means the latest version of the chart will be installed
	viper.SetDefault(LoggingOperatorChartVersion, "")
	viper.SetDefault(LoggingOperatorImageTag, "0.0.5")

	viper.BindEnv(ControlPlaneNamespace, "KUBERNETES_NAMESPACE")
	viper.SetDefault(ControlPlaneNamespace, "default")

	viper.SetDefault(PipelineSystemNamespace, "pipeline-system")
	viper.SetDefault(EksTemplateLocation, filepath.Join(pwd, "templates", "eks"))
	viper.SetDefault(EksASGFulfillmentTimeout, "10m")

	viper.SetDefault(SpotguideAllowPrereleases, false)
	viper.SetDefault(SpotguideAllowPrivateRepos, false)
	viper.SetDefault(SpotguideSyncInterval, 5*time.Minute)
	viper.SetDefault(SpotguideSharedLibraryGitHubOrganization, "spotguides")

	viper.SetDefault("issue.type", "github")
	viper.SetDefault("issue.githubLabels", []string{"community"})
	viper.SetDefault("issue.githubOwner", "banzaicloud")
	viper.SetDefault("issue.githubRepository", "pipeline")

	viper.SetDefault("cert.source", "file")
	viper.SetDefault("cert.path", "config/certs")

	viper.SetDefault("gitlab.baseURL", "https://gitlab.com/")

	viper.SetDefault(IstioOperatorChartVersion, "0.0.2")
	viper.SetDefault(IstioGrafanaDashboardLocation, filepath.Join(pwd, "etc", "dashboards", "istio"))

	viper.SetDefault(NodePoolLabelSetOperatorChartVersion, "0.0.2")

	viper.SetDefault(PipelineLabelDomain, "banzaicloud.io")
	viper.SetDefault(ForbiddenLabelDomains, []string{
		"k8s.io",
		"kubernetes.io",
		"google.com",
	})

	// Cadence config
	viper.SetDefault("cadence.port", 7933)
	viper.SetDefault("cadence.domain", "pipeline")

	viper.SetDefault(AmazonInitializeRegionKey, "us-west-1")
	viper.SetDefault(AlibabaInitializeRegionKey, "eu-central-1")

	// Prometheus service defaults
	viper.SetDefault(PrometheusServiceName, "monitor-prometheus-server")
	viper.SetDefault(PrometheusServiceContext, "prometheus")
	viper.SetDefault(PrometheusLocalPort, 9090)

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	// Confirm which config file is used
	fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	viper.SetEnvPrefix("pipeline")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)
}

//GetCORS gets CORS related config
func GetCORS() cors.Config {
	viper.SetDefault("cors.AllowAllOrigins", true)
	viper.SetDefault("cors.AllowOrigins", []string{})
	viper.SetDefault("cors.AllowOriginsRegexp", "")
	viper.SetDefault("cors.AllowMethods", []string{"PUT", "DELETE", "GET", "POST", "OPTIONS", "PATCH"})
	viper.SetDefault("cors.AllowHeaders", []string{"Origin", "Authorization", "Content-Type", "secretId"})
	viper.SetDefault("cors.ExposeHeaders", []string{"Content-Length"})
	viper.SetDefault("cors.AllowCredentials", true)
	viper.SetDefault("cors.MaxAge", 12)

	config := cors.DefaultConfig()
	config.AllowAllOrigins = viper.GetBool("cors.AllowAllOrigins")
	if !config.AllowAllOrigins {
		allowOriginsRegexp := viper.GetString("cors.AllowOriginsRegexp")
		if allowOriginsRegexp != "" {
			originsRegexp, err := regexp.Compile(fmt.Sprintf("^(%s)$", allowOriginsRegexp))
			if err == nil {
				config.AllowOriginFunc = func(origin string) bool {
					return originsRegexp.Match([]byte(origin))
				}
			}
		} else if allowOrigins := viper.GetStringSlice("cors.AllowOrigins"); len(allowOrigins) > 0 {
			config.AllowOrigins = allowOrigins
		}
	}

	config.AllowMethods = viper.GetStringSlice("cors.AllowMethods")
	config.AllowHeaders = viper.GetStringSlice("cors.AllowHeaders")
	config.ExposeHeaders = viper.GetStringSlice("cors.ExposeHeaders")
	config.AllowCredentials = viper.GetBool("cors.AllowCredentials")
	maxAge := viper.GetInt("cors.MaxAge")
	config.MaxAge = time.Duration(maxAge) * time.Hour
	return config
}

// GetStateStorePath returns the state store path
func GetStateStorePath(clusterName string) string {
	stateStorePath := viper.GetString("statestore.path")
	if len(clusterName) == 0 {
		return stateStorePath
	}

	return fmt.Sprintf("%s/%s", stateStorePath, clusterName)
}

// GetHelmPath returns local helm path
func GetHelmPath(organizationName string) string {
	return fmt.Sprintf("%s/%s", viper.GetString(helmPath), organizationName)
}
