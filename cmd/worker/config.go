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
	"fmt"
	"os"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/pkg/viperx"
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

	// Error handling configuration
	Errors errorhandler.Config

	// Pipeline configuration
	Pipeline PipelineConfig

	// Auth configuration
	Auth authConfig

	// Cluster configuration
	Cluster cmd.ClusterConfig

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

	if err := c.Errors.Validate(); err != nil {
		return err
	}

	if err := c.Pipeline.Validate(); err != nil {
		return err
	}

	if err := c.Auth.Validate(); err != nil {
		return err
	}

	if err := c.Cluster.Validate(); err != nil {
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

// Process post-processes the configuration after loading (before validation).
func (c *configuration) Process() error {
	if err := c.Cluster.Process(); err != nil {
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

// authConfig contains auth configuration.
type authConfig struct {
	Token authTokenConfig
}

// Validate validates the configuration.
func (c authConfig) Validate() error {
	if err := c.Token.Validate(); err != nil {
		return err
	}

	return nil
}

// authTokenConfig contains auth configuration.
type authTokenConfig struct {
	SigningKey string
	Issuer     string
	Audience   string
}

// Validate validates the configuration.
func (c authTokenConfig) Validate() error {
	if c.SigningKey == "" {
		return errors.New("auth token signing key is required")
	}

	if len(c.SigningKey) < 32 {
		return errors.New("auth token signing key must be at least 32 characters")
	}

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

	// Global configuration
	v.SetDefault("environment", "production")
	v.SetDefault("debug", false)
	v.SetDefault("shutdownTimeout", 15*time.Second)

	// ErrorHandler configuration
	v.Set("errors.serviceName", appName)
	v.Set("errors.serviceVersion", version)

	// Pipeline configuration
	v.SetDefault("pipeline.basePath", "")

	// Load common configuration
	cmd.Configure(v, p)

	// Auth configuration
	v.SetDefault("auth.token.issuer", "https://banzaicloud.com/")
	v.SetDefault("auth.token.audience", "https://pipeline.banzaicloud.com")

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
	v.SetDefault("cadence.createNonexistentDomain", false)
	v.SetDefault("cadence.workflowExecutionRetentionPeriodInDays", 3)

	// OIDC configuration
	viper.SetDefault("auth.dexURL", "http://127.0.0.1:5556/dex")
	viper.RegisterAlias("auth.oidcIssuerURL", "auth.dexURL")
	viper.SetDefault("auth.dexInsecure", false)
	viper.RegisterAlias("auth.oidcIssuerInsecure", "auth.dexInsecure")
	viper.SetDefault("auth.dexGrpcAddress", "127.0.0.1:5557")
	viper.SetDefault("auth.dexGrpcCaCert", "")
}

func registerAliases(v *viper.Viper) {
	// Auth configuration
	viperx.RegisterAlias(v, "auth.tokensigningkey", "auth.token.signingKey")
	viperx.RegisterAlias(v, "auth.jwtissuer", "auth.token.issuer")
	viperx.RegisterAlias(v, "auth.jwtaudience", "auth.token.audience")
}
