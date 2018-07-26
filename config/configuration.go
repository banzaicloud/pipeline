package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/spf13/viper"
)

const (
	// local helm path
	helmPath = "helm.path"
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
	viper.SetDefault("helm.tillerVersion", "v2.9.0")
	viper.SetDefault("helm.stableRepositoryURL", "https://kubernetes-charts.storage.googleapis.com")
	viper.SetDefault("helm.banzaiRepositoryURL", "http://kubernetes-charts.banzaicloud.com")
	viper.SetDefault(helmPath, "./orgs")
	viper.SetDefault("cloud.defaultProfileName", "default")
	viper.SetDefault("cloud.configRetryCount", 30)
	viper.SetDefault("cloud.configRetrySleep", 15)
	viper.SetDefault("aws.credentials.path", "secret/data/banzaicloud/aws")
	viper.SetDefault("logging.kubicornloglevel", "debug")
	viper.SetDefault("catalog.repositoryUrl", "http://kubernetes-charts.banzaicloud.com/branch/spotguide")

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err.Error())
	}
	viper.SetDefault("statestore.path", fmt.Sprintf("%s/statestore/", pwd))

	viper.SetDefault("auth.jwtissuer", "https://banzaicloud.com/")
	viper.SetDefault("auth.jwtaudience", "https://pipeline.banzaicloud.com")

	viper.SetDefault("pipeline.listenport", 9090)
	viper.SetDefault("pipeline.certfile", "")
	viper.SetDefault("pipeline.keyfile", "")
	viper.SetDefault("pipeline.uipath", "/ui")
	viper.SetDefault("pipeline.basepath", "")
	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("metrics.path", "/metrics")
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

	viper.SetDefault("dns.domain", "banzaicloud.io")
	viper.SetDefault("dns.secretNamespace", "pipeline-infra")
	viper.SetDefault("dns.gcIntervalMinute", 1)
	viper.SetDefault("route53.maintenanceWindowMinute", 15)

	ReleaseName := os.Getenv("KUBERNETES_RELEASE_NAME")
	if ReleaseName == "" {
		ReleaseName = "pipeline"
	}
	viper.SetDefault("monitor.release", ReleaseName)
	viper.SetDefault("monitor.enabled", false)
	viper.SetDefault("monitor.configmap", "")
	viper.SetDefault("monitor.mountpath", "")
	viper.SetDefault("monitor.grafanaAdminUsername", "admin")

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
	viper.SetDefault("cors.AllowOrigins", []string{"http://", "https://"})
	viper.SetDefault("cors.AllowMethods", []string{"PUT", "DELETE", "GET", "POST", "OPTIONS"})
	viper.SetDefault("cors.AllowHeaders", []string{"Origin", "Authorization", "Content-Type", "secretId"})
	viper.SetDefault("cors.ExposeHeaders", []string{"Content-Length"})
	viper.SetDefault("cors.AllowCredentials", true)
	viper.SetDefault("cors.MaxAge", 12)

	config := cors.DefaultConfig()
	cors.DefaultConfig()
	config.AllowAllOrigins = viper.GetBool("cors.AllowAllOrigins")
	if !config.AllowAllOrigins {
		config.AllowOrigins = viper.GetStringSlice("cors.AllowOrigins")
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
