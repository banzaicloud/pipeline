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

package main

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Log configuration
	Log log.Config

	// ErrorHandler configuration
	ErrorHandler errorhandler.Config
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if err := c.ErrorHandler.Validate(); err != nil {
		return err
	}

	return nil
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, _ *pflag.FlagSet) {
	// Application constants
	v.Set("appName", appName)
	v.Set("appVersion", version)

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	// Log configuration
	v.SetDefault("logging.logformat", "text")
	v.SetDefault("logging.loglevel", "debug")
	v.RegisterAlias("log.format", "logging.logformat") // TODO: deprecate the above
	v.RegisterAlias("log.level", "logging.loglevel")
	v.RegisterAlias("log.noColor", "no_color")

	// ErrorHandler configuration
	v.RegisterAlias("errorHandler.serviceName", "appName")
	v.RegisterAlias("errorHandler.serviceVersion", "appVersion")
}
