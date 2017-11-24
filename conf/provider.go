package conf

import (
	"github.com/spf13/viper"
)

//Database initializes the database config
func Provider() string {

	provider := viper.GetString("dev.cloudprovider")
	if provider == "" {
		panic("Provider not set")
	}

	return provider

}
