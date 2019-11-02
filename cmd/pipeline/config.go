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
	"time"

	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/app/frontend"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
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

	// Auth configuration
	Auth auth.Config

	// Cadence configuration
	Cadence cadence.Config

	// Frontend configuration
	Frontend frontend.Config

	// Cluster configuration
	Cluster cmd.ClusterConfig

	Cloudinfo struct {
		Endpoint string
	}

	Github struct {
		Token string
	}

	SpotMetrics struct {
		Enabled            bool
		CollectionInterval time.Duration
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

	return nil
}

// Process post-processes the configuration after loading (before validation).
func (c *configuration) Process() error {
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
	// ErrorHandler configuration
	v.Set("errors.serviceName", appName)
	v.Set("errors.serviceVersion", version)

	// Telemetry configuration
	v.SetDefault("telemetry.enabled", false)
	p.String("telemetry-addr", "127.0.0.1:9900", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.addr", p.Lookup("telemetry-addr"))
	v.SetDefault("telemetry.addr", "127.0.0.1:9900")
	v.SetDefault("telemetry.debug", true)

	// Load common configuration
	cmd.Configure(v, p)

	// Auth configuration
	v.SetDefault("auth.role.default", auth.RoleAdmin)
	v.SetDefault("auth.role.binding", map[string]string{
		auth.RoleAdmin:  ".*",
		auth.RoleMember: "",
	})

	v.SetDefault("frontend.issue.enabled", false)
	v.SetDefault("frontend.issue.driver", "github")
	v.SetDefault("frontend.issue.labels", []string{"community"})

	v.SetDefault("frontend.issue.github.token", "")
	v.SetDefault("frontend.issue.github.owner", "banzaicloud")
	v.SetDefault("frontend.issue.github.repository", "pipeline-issues")

	v.SetDefault("spotmetrics.enabled", false)
	v.SetDefault("spotmetrics.collectionInterval", 30*time.Second)
}
