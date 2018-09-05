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

	// DNSSecretNamespace configuration key for the K8s namespace setting
	// external DNS services secrets are mounted to.
	DNSSecretNamespace = "dns.secretNamespace"

	// DNSGcIntervalMinute configuration key for the interval setting at which the DNS garbage collector runs
	DNSGcIntervalMinute = "dns.gcIntervalMinute"

	// DNSGcLogLevel configuration key for the DNS garbage collector logging level default value: "debug"
	DNSGcLogLevel = "dns.gcLogLevel"

	// DNSExternalDnsChartVersion set the external-dns chart version default value: "0.5.4"
	DNSExternalDnsChartVersion = "dns.externalDnsChartVersion"

	// Route53MaintenanceWndMinute configuration key for the maintenance window for Route53.
	// This is the maintenance window before the next AWS Route53 pricing period starts
	Route53MaintenanceWndMinute = "route53.maintenanceWindowMinute"

	//PipelineMonitorNamespace pipeline infra namespace key
	PipelineMonitorNamespace = "infra.namespace"

	// EksTemplateLocation is the configuration key the location to get EKS Cloud Formation templates from
	// the location to get EKS Cloud Formation templates from
	EksTemplateLocation = "eks.templateLocation"

	// AwsCredentialPath is the path in Vault to get AWS credentials from for Pipeline
	AwsCredentialPath = "aws.credentials.path"

	// Config keys to GKE resource delete
	GKEResourceDeleteWaitAttempt  = "gke.resourceDeleteWaitAttempt"
	GKEResourceDeleteSleepSeconds = "gke.resourceDeleteSleepSeconds"

	// Config keys to OKE nodepool wait
	OKEWaitAttemptsForNodepoolActive = "oke.waitAttemptsForNodepoolActive"
	OKESleepSecondsForNodepoolActive = "oke.sleepSecondsForNodepoolActive"
)

//Init initializes the configurations
func init() {

	viper.AddConfigPath("$HOME/config")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$PIPELINE_CONFIG_DIR/")

	viper.SetConfigName("config")
	//viper.SetConfigType("toml")

	// Set defaults TODO expand defaults
	viper.SetDefault("drone.url", "http://localhost:8000")
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
	viper.SetDefault("logging.kubicornloglevel", "debug")

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err.Error())
	}
	viper.SetDefault("statestore.path", fmt.Sprintf("%s/statestore/", pwd))

	viper.SetDefault("auth.jwtissuer", "https://banzaicloud.com/")
	viper.SetDefault("auth.jwtaudience", "https://pipeline.banzaicloud.com")
	viper.SetDefault("auth.secureCookie", true)

	viper.SetDefault("pipeline.listenport", 9090)
	viper.SetDefault("pipeline.certfile", "")
	viper.SetDefault("pipeline.keyfile", "")
	viper.SetDefault("pipeline.uipath", "/ui")
	viper.SetDefault("pipeline.basepath", "")
	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("metrics.port", ":9900")
	viper.SetDefault("database.dialect", "mysql")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.user", "kellyslater")
	viper.SetDefault("database.password", "pipemaster123!")
	viper.SetDefault("database.dbname", "pipelinedb")
	viper.SetDefault("database.logging", false)
	viper.SetDefault("audit.enabled", true)
	viper.SetDefault("audit.headers", []string{"secretId"})
	viper.SetDefault("audit.skippaths", []string{"/auth/github/callback", "/pipeline/api"})
	viper.SetDefault("tls.validity", "8760h") // 1 year
	viper.SetDefault(DNSBaseDomain, "banzaicloud.io")
	viper.SetDefault(DNSSecretNamespace, "pipeline-infra")
	viper.SetDefault(DNSGcIntervalMinute, 1)
	viper.SetDefault(DNSExternalDnsChartVersion, "0.7.5")
	viper.SetDefault(DNSGcLogLevel, "debug")
	viper.SetDefault(Route53MaintenanceWndMinute, 15)

	viper.SetDefault(GKEResourceDeleteWaitAttempt, 12)
	viper.SetDefault(GKEResourceDeleteSleepSeconds, 5)

	viper.SetDefault(OKEWaitAttemptsForNodepoolActive, 60)
	viper.SetDefault(OKESleepSecondsForNodepoolActive, 30)

	ReleaseName := os.Getenv("KUBERNETES_RELEASE_NAME")
	if ReleaseName == "" {
		ReleaseName = "pipeline"
	}
	viper.SetDefault("monitor.release", ReleaseName)
	viper.SetDefault("monitor.enabled", false)
	viper.SetDefault("monitor.configmap", "")
	viper.SetDefault("monitor.mountpath", "")
	viper.SetDefault("monitor.grafanaAdminUsername", "admin")

	viper.SetDefault(PipelineMonitorNamespace, "pipeline-infra")
	viper.SetDefault(EksTemplateLocation, filepath.Join(pwd, "templates", "eks"))

	// Cadence config
	viper.SetDefault("cadence.port", 7933)
	viper.SetDefault("cadence.domain", "banzaicloud")

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	// Confirm which config file is used
	fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	viper.SetEnvPrefix("pipeline")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

//GetCORS gets CORS related config
func GetCORS() cors.Config {
	viper.SetDefault("cors.AllowAllOrigins", true)
	viper.SetDefault("cors.AllowOrigins", []string{})
	viper.SetDefault("cors.AllowOriginsRegexp", "")
	viper.SetDefault("cors.AllowMethods", []string{"PUT", "DELETE", "GET", "POST", "OPTIONS"})
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
