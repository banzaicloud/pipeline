package config

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/spf13/viper"
	"log"
	"strings"
	"time"
)

//Init initializes the configurations
func Init() {

	viper.AddConfigPath("$HOME/pipeline")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./config")

	viper.SetConfigName("config")
	//viper.SetConfigType("toml")

	// Set defaults TODO expand defaults
	viper.SetDefault("helm.retryAttempt", 30)
	viper.SetDefault("helm.retrySleepSeconds", 15)
	viper.SetDefault("helm.stableRepositoryURL", "https://kubernetes-charts.storage.googleapis.com")
	viper.SetDefault("helm.banzaiRepositoryURL", "http://kubernetes-charts.banzaicloud.com")
	viper.SetDefault("cloud.gkeCredentialPath", "./conf/gke_credential.json")
	viper.SetDefault("cloud.defaultProfileName", "default")
	viper.SetDefault("logger.kubicornloglevel", 4)

	viper.SetDefault("statestore.path", "./statestore")

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

func GetCORS() cors.Config {
	viper.SetDefault("cors.AllowAllOrigins", true)
	viper.SetDefault("cors.AllowOrigins", []string{"http://", "https://"})
	viper.SetDefault("cors.AllowMethods", []string{"PUT", "DELETE", "GET", "POST"})
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
