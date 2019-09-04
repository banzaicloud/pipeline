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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality
	Debug bool

	// Timeout for graceful shutdown
	ShutdownTimeout time.Duration

	// Log configuration
	Log log.Config

	// ErrorHandler configuration
	ErrorHandler errorhandler.Config

	// Pipeline configuration
	Pipeline PipelineConfig

	// Database connection information
	Database database.Config

	// Cadence configuration
	Cadence cadence.Config
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if c.Environment == "" {
		return errors.New("environment is required")
	}

	if err := c.ErrorHandler.Validate(); err != nil {
		return err
	}

	if err := c.Pipeline.Validate(); err != nil {
		return err
	}

	if err := c.Database.Validate(); err != nil {
		return err
	}

	if err := c.Cadence.Validate(); err != nil {
		return err
	}

	return nil
}

// PipelineConfig contains application specific config.
type PipelineConfig struct {
	BasePath string
}

// Validate validates the configuration.
func (c PipelineConfig) Validate() error {
	return nil
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, p *pflag.FlagSet) {
	v.AllowEmptyEnv(true)
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath(fmt.Sprintf("$%s_CONFIG_DIR/", strings.ToUpper(envPrefix)))
	p.Init(friendlyAppName, pflag.ExitOnError)
	pflag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", friendlyAppName)
		pflag.PrintDefaults()
	}
	_ = v.BindPFlags(p)

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Application constants
	v.Set("appName", appName)
	v.Set("appVersion", version)

	// Global configuration
	v.SetDefault("environment", "production")
	v.SetDefault("debug", false)
	v.SetDefault("shutdownTimeout", 15*time.Second)
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

	// Pipeline configuration
	v.SetDefault("pipeline.basePath", "")

	// Database configuration
	v.SetDefault("database.dialect", "mysql")
	_ = v.BindEnv("database.host")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.tls", "")
	_ = v.BindEnv("database.user")
	_ = v.BindEnv("database.password")
	v.RegisterAlias("database.pass", "database.password") // TODO: deprecate password
	_ = v.BindEnv("database.dbname")
	v.RegisterAlias("database.name", "database.dbname") // TODO: deprecate dbname
	v.SetDefault("database.params", map[string]string{
		"charset": "utf8mb4",
	})
	v.RegisterAlias("database.enableLog", "database.logging")

	// Cadence configuration
	_ = v.BindEnv("cadence.host")
	v.SetDefault("cadence.port", 7933)
	v.SetDefault("cadence.domain", "pipeline")
	v.SetDefault("cadence.createNonexistentDomain", false)
	v.SetDefault("cadence.workflowExecutionRetentionPeriodInDays", 3)

	// Spotguide configuration
	_ = v.BindEnv("github.token")
	_ = v.BindEnv("gitlab.token")
	v.SetDefault("cicd.scm", "github")
	v.SetDefault("spotguide.syncInterval", 15*time.Minute)
	v.SetDefault("spotguide.allowPrereleases", false)
	v.SetDefault("spotguide.allowPrivateRepos", false)
	v.SetDefault("spotguide.sharedLibraryGitHubOrganization", "spotguides")

	// OIDC configuration
	viper.SetDefault("auth.dexURL", "http://127.0.0.1:5556/dex")
	viper.RegisterAlias("auth.oidcIssuerURL", "auth.dexURL")
	viper.SetDefault("auth.dexInsecure", false)
	viper.RegisterAlias("auth.oidcIssuerInsecure", "auth.dexInsecure")
	viper.SetDefault("auth.dexGrpcAddress", "127.0.0.1:5557")
	viper.SetDefault("auth.dexGrpcCaCert", "")
}
