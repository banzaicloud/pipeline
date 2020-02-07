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
	"github.com/banzaicloud/pipeline/src/auth"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	cmd.Config `mapstructure:",squash"`

	Audit struct {
		Enabled   bool
		Headers   []string
		SkipPaths []string
	}

	CORS struct {
		AllowAllOrigins    bool
		AllowOrigins       []string
		AllowOriginsRegexp string
	}

	// Frontend configuration
	Frontend frontend.Config

	Pipeline PipelineConfig

	SpotMetrics struct {
		Enabled            bool
		CollectionInterval time.Duration
	}
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	return errors.Combine(c.Config.Validate(), c.Frontend.Validate())
}

// Process post-processes the configuration after loading (before validation).
func (c *configuration) Process() error {
	var err error

	err = errors.Append(err, c.Config.Process())

	return err
}

type PipelineConfig struct {
	Addr         string
	InternalAddr string
	BasePath     string
	CertFile     string
	KeyFile      string
	UUID         string
	External     struct {
		URL      string
		Insecure bool
	}
}

func (c PipelineConfig) Validate() error {
	var err error

	if c.Addr == "" {
		err = errors.Append(err, errors.New("pipeline address is required"))
	}

	if c.InternalAddr == "" {
		err = errors.Append(err, errors.New("pipeline internal address is required"))
	}

	return err
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
	v.SetDefault("pipeline::uuid", "")
	v.SetDefault("pipeline::external::url", "")
	v.SetDefault("pipeline::external::insecure", false)

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

	v.SetDefault("spotmetrics::enabled", false)
	v.SetDefault("spotmetrics::collectionInterval", 30*time.Second)

	v.SetDefault("audit::enabled", true)
	v.SetDefault("audit::headers", []string{"secretId"})
	v.SetDefault("audit::skipPaths", []string{"/auth/dex/callback", "/pipeline/api"})
}
