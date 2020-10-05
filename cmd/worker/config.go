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
	"github.com/banzaicloud/pipeline/src/auth"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	cmd.Config `mapstructure:",squash"`

	Auth authConfig

	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality
	Debug bool

	// Timeout for graceful shutdown
	ShutdownTimeout time.Duration

	// TODO: remove if not required
	// This is required by the global config, so it's hard to determine whether
	// it's really required here (i.e. used through global config that's
	// initialized from this).
	Pipeline struct {
		Enterprise bool
		External   struct {
			URL      string
			Insecure bool
		}
		UUID string
	}
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	var errs error

	errs = errors.Append(errs, c.Auth.Validate())
	errs = errors.Append(errs, c.Config.Validate())

	if c.Environment == "" {
		errs = errors.Append(errs, errors.New("environment is required"))
	}

	// TODO: this config is only used here, so the validation is here too. Either the config or the validation should be moved somewhere else.
	if c.Distribution.PKE.Amazon.GlobalRegion == "" {
		errs = errors.Append(errs, errors.New("pke amazon global region is required"))
	}

	return errs
}

// Process post-processes the configuration after loading (before validation).
func (c *configuration) Process() error {
	var err error

	err = errors.Append(err, c.Config.Process())

	return err
}

type authConfig struct {
	// TODO: remove this when the global config no longer needs them
	Cookie struct {
		Secure    bool
		SetDomain bool
	}
	// TODO: remove this when the global config no longer needs them
	OIDC struct {
		Issuer string
	}
	Token auth.TokenConfig
}

func (c authConfig) Validate() error {
	var errs error

	if c.OIDC.Issuer == "" {
		errs = errors.Append(errs, errors.New("auth oidc issuer is required"))
	}

	errs = errors.Append(errs, c.Token.Validate())

	return errs
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

	// Global configuration
	v.SetDefault("environment", "production")
	v.SetDefault("debug", false)
	v.SetDefault("shutdownTimeout", 15*time.Second)

	// Cadence configuration
	v.SetDefault("cadence::createNonexistentDomain", false)
	v.SetDefault("cadence::workflowExecutionRetentionPeriodInDays", 3)

	v.SetDefault("pipeline::uuid", "")
	v.SetDefault("pipeline::enterprise", false)
	v.SetDefault("pipeline::external::url", "")
	v.SetDefault("pipeline::external::insecure", false)
}
