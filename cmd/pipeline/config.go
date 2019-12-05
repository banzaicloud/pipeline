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

	"github.com/banzaicloud/pipeline/internal/app/frontend"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/src/auth"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Log configuration
	Log log.Config

	// Error handling configuration
	Errors errorhandler.Config

	// Telemetry configuration
	Telemetry telemetryConfig

	Pipeline struct {
		Addr         string
		InternalAddr string
		BasePath     string
		CertFile     string
		KeyFile      string
	}

	// Auth configuration
	Auth auth.Config

	// Database configuration
	Database struct {
		database.Config `mapstructure:",squash"`

		AutoMigrate bool
	}

	// Cadence configuration
	Cadence cadence.Config

	CORS struct {
		AllowAllOrigins    bool
		AllowOrigins       []string
		AllowOriginsRegexp string
	}

	// Frontend configuration
	Frontend frontend.Config

	// Cluster configuration
	Cluster cmd.ClusterConfig

	Cloudinfo struct {
		Endpoint string
	}

	CICD struct {
		Enabled  bool
		Database database.Config
	}

	Github struct {
		Token string
	}

	SpotMetrics struct {
		Enabled            bool
		CollectionInterval time.Duration
	}

	Audit struct {
		Enabled   bool
		Headers   []string
		SkipPaths []string
	}
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if err := c.Errors.Validate(); err != nil {
		return err
	}

	if err := c.Telemetry.Validate(); err != nil {
		return err
	}

	if c.Pipeline.Addr == "" {
		return errors.New("pipeline address is required")
	}
	if c.Pipeline.InternalAddr == "" {
		return errors.New("pipeline internal address is required")
	}

	if err := c.Auth.Validate(); err != nil {
		return err
	}

	if err := c.Cadence.Validate(); err != nil {
		return err
	}

	if err := c.Frontend.Validate(); err != nil {
		return err
	}

	if err := c.Cluster.Validate(); err != nil {
		return err
	}

	if c.Cloudinfo.Endpoint == "" {
		return errors.New("cloudinfo endpoint is required")
	}

	// if c.CICD.Enabled {
	if err := c.CICD.Database.Validate(); err != nil {
		return err
	}
	// }

	return nil
}

// Process post-processes the configuration after loading (before validation).
func (c *configuration) Process() error {
	if err := c.Auth.Process(); err != nil {
		return err
	}

	if err := c.Cluster.Process(); err != nil {
		return err
	}

	if c.Frontend.Issue.Github.Token == "" {
		c.Frontend.Issue.Github.Token = c.Github.Token
	}

	return nil
}

// telemetryConfig contains telemetry configuration.
type telemetryConfig struct {
	Enabled bool
	Addr    string
	Debug   bool
}

// Validate validates the configuration.
func (c telemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Addr == "" {
		return errors.New("telemetry http server address is required")
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
	v.SetEnvKeyReplacer(strings.NewReplacer("::", "_", ".", "_", "-", "_"))
	v.AutomaticEnv()

	v.SetKeyDelimiter("::")

	// Load common configuration
	cmd.Configure(v, p)

	// ErrorHandler configuration
	v.Set("errors::serviceName", appName)
	v.Set("errors::serviceVersion", version)

	// Telemetry configuration
	v.SetDefault("telemetry::enabled", false)
	p.String("telemetry-addr", "127.0.0.1:9900", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry::addr", p.Lookup("telemetry-addr"))
	v.SetDefault("telemetry::addr", "127.0.0.1:9900")
	v.SetDefault("telemetry::debug", true)

	// Pipeline configuration
	p.String("addr", "127.0.0.1:9090", "Pipeline HTTP server address")
	_ = v.BindPFlag("pipeline::addr", p.Lookup("addr"))
	v.SetDefault("pipeline::addr", "127.0.0.1:9090")
	v.SetDefault("pipeline::internalAddr", "127.0.0.1:9091")
	v.SetDefault("pipeline::basePath", "")
	v.SetDefault("pipeline::certFile", "")
	v.SetDefault("pipeline::keyFile", "")

	// Auth configuration
	v.SetDefault("auth::redirectUrl::login", "/ui")
	v.SetDefault("auth::redirectUrl::signup", "/ui")

	v.SetDefault("auth::role::default", auth.RoleAdmin)
	v.SetDefault("auth::role::binding", map[string]string{
		auth.RoleAdmin:  ".*",
		auth.RoleMember: "",
	})

	// Database config
	v.SetDefault("database::autoMigrate", false)

	v.SetDefault("cors::allowAllOrigins", true)
	v.SetDefault("cors::allowOrigins", []string{})
	v.SetDefault("cors::allowOriginsRegexp", "")

	v.SetDefault("frontend::issue::enabled", false)
	v.SetDefault("frontend::issue::driver", "github")
	v.SetDefault("frontend::issue::labels", []string{"community"})

	v.SetDefault("frontend::issue::github::token", "")
	v.SetDefault("frontend::issue::github::owner", "banzaicloud")
	v.SetDefault("frontend::issue::github::repository", "pipeline-issues")

	v.SetDefault("spotmetrics::enabled", false)
	v.SetDefault("spotmetrics::collectionInterval", 30*time.Second)

	v.SetDefault("audit::enabled", true)
	v.SetDefault("audit::headers", []string{"secretId"})
	v.SetDefault("audit::skipPaths", []string{"/auth/dex/callback", "/pipeline/api"})
}
