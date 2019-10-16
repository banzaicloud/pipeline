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

	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/app/frontend"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/pkg/viperx"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Log configuration
	Log log.Config

	// ErrorHandler configuration
	ErrorHandler errorhandler.Config

	// Auth configuration
	Auth authConfig

	// Frontend configuration
	Frontend frontend.Config

	// Anchore default configuration
	Anchore anchore.Config
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if err := c.ErrorHandler.Validate(); err != nil {
		return err
	}

	if err := c.Auth.Validate(); err != nil {
		return err
	}

	if err := c.Frontend.Validate(); err != nil {
		return err
	}

	if err := c.Anchore.Validate(); err != nil {
		return err
	}

	return nil
}

// authConfig contains auth configuration.
type authConfig struct {
	Token authTokenConfig
	Role  authRoleConfig
}

// Validate validates the configuration.
func (c authConfig) Validate() error {
	if err := c.Token.Validate(); err != nil {
		return err
	}

	if err := c.Role.Validate(); err != nil {
		return err
	}

	return nil
}

// authRoleConfig contains role based authorization configuration.
type authRoleConfig struct {
	Default string
	Binding map[string]string
}

// Validate validates the configuration.
func (c authRoleConfig) Validate() error {
	if c.Default == "" {
		return errors.New("auth role default is required")
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

	// Auth configuration
	v.SetDefault("auth.token.issuer", "https://banzaicloud.com/")
	v.SetDefault("auth.token.audience", "https://pipeline.banzaicloud.com")

	v.SetDefault("auth.role.default", auth.RoleAdmin)
	v.SetDefault("auth.role.binding", map[string]string{
		auth.RoleAdmin:  ".*",
		auth.RoleMember: "",
	})

	v.SetDefault("frontend.issue.enabled", false)
	v.SetDefault("frontend.issue.driver", "github")
	v.SetDefault("frontend.issue.labels", []string{"community"})

	v.RegisterAlias("frontend.issue.github.token", "github.token")
	v.SetDefault("frontend.issue.github.owner", "banzaicloud")
	v.SetDefault("frontend.issue.github.repository", "pipeline-issues")

	v.SetDefault("anchore.apiEnabled", true)
	v.SetDefault("anchore.enabled", false)
	v.SetDefault("anchore.endpoint", "")
	v.SetDefault("anchore.adminuser", "")
	v.SetDefault("anchore.adminpass", "")
}

func registerAliases(v *viper.Viper) {
	// Auth configuration
	viperx.RegisterAlias(v, "auth.tokensigningkey", "auth.token.signingKey")
	viperx.RegisterAlias(v, "auth.jwtissuer", "auth.token.issuer")
	viperx.RegisterAlias(v, "auth.jwtaudience", "auth.token.audience")

	// Frontend configuration
	viperx.RegisterAlias(v, "issue.type", "frontend.issue.driver")
	viperx.RegisterAlias(v, "issue.githubLabels", "frontend.issue.labels")

	viperx.RegisterAlias(v, "issue.githubOwner", "frontend.issue.github.owner")
	viperx.RegisterAlias(v, "issue.githubRepository", "frontend.issue.github.repository")
}
