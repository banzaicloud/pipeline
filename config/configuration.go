package config

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"time"
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
	viper.SetDefault("drone.enabled", false)
	viper.SetDefault("auth.vaultpath", "./vault")
	viper.SetDefault("helm.retryAttempt", 30)
	viper.SetDefault("helm.retrySleepSeconds", 15)
	viper.SetDefault("helm.stableRepositoryURL", "https://kubernetes-charts.storage.googleapis.com")
	viper.SetDefault("helm.banzaiRepositoryURL", "http://kubernetes-charts.banzaicloud.com")
	viper.SetDefault("cloud.gkeCredentialPath", "./conf/gke_credential.json")
	viper.SetDefault("cloud.defaultProfileName", "default")
	viper.SetDefault("cloud.configRetryCount", 30)
	viper.SetDefault("cloud.configRetrySleep", 15)
	viper.SetDefault("logging.kubicornloglevel", "debug")
	viper.SetDefault("statestore.path", "./statestore")
	viper.SetDefault("statestore.configmap", "")
	viper.SetDefault("pipeline.listenport", 9090)
	viper.SetDefault("database.dialect", "mysql")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.user", "kellyslater")
	viper.SetDefault("database.password", "pipemaster123!")
	viper.SetDefault("database.dbname", "pipelinedb")

	ReleaseName := os.Getenv("KUBERNETES_RELEASE_NAME")
	if ReleaseName == "" {
		ReleaseName = "pipeline"
	}
	viper.SetDefault("monitor.release", ReleaseName)
	viper.SetDefault("monitor.enabled", false)
	viper.SetDefault("monitor.configmap", "")

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
	viper.SetDefault("cors.AllowHeaders", []string{"Origin", "Authorization", "Content-Type"})
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
