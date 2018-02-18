package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"strings"
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
