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

package auth

import (
	"emperror.dev/errors"
)

// Config contains auth configuration.
type Config struct {
	OIDC        OIDCConfig
	CLI         CLIConfig
	RedirectURL RedirectURLConfig
	Cookie      CookieConfig
	Token       TokenConfig
	Role        RoleConfig
}

// Validate validates the configuration.
func (c Config) Validate() error {
	return errors.Combine(
		c.OIDC.Validate(),
		c.CLI.Validate(),
		c.RedirectURL.Validate(),
		c.Cookie.Validate(),
		c.Token.Validate(),
		c.Role.Validate(),
	)
}

// Process post-processes the configuration after loading (before validation).
func (c *Config) Process() error {
	if err := c.RedirectURL.Process(); err != nil {
		return err
	}

	return nil
}

// OIDCConfig contains OIDC auth configuration.
type OIDCConfig struct {
	Issuer       string
	Insecure     bool
	ClientID     string
	ClientSecret string
}

// Validate validates the configuration.
func (c OIDCConfig) Validate() error {
	var err error

	if c.Issuer == "" {
		err = errors.Append(err, errors.New("auth oidc issuer is required"))
	}

	if c.ClientID == "" {
		err = errors.Append(err, errors.New("auth oidc client ID is required"))
	}

	if c.ClientSecret == "" {
		err = errors.Append(err, errors.New("auth oidc client secret is required"))
	}

	return err
}

// CLIConfig contains cli auth configuration.
type CLIConfig struct {
	ClientID string
}

// Validate validates the configuration.
func (c CLIConfig) Validate() error {
	if c.ClientID == "" {
		return errors.New("auth cli client ID is required")
	}

	return nil
}

// RedirectURLConfig contains the URLs the user is redirected to after certain authentication events.
type RedirectURLConfig struct {
	Login  string
	Signup string
}

// Validate validates the configuration.
func (c RedirectURLConfig) Validate() error {
	var err error

	if c.Login == "" {
		err = errors.Append(err, errors.New("auth login redirect URL is required"))
	}

	if c.Signup == "" {
		err = errors.Append(err, errors.New("auth signup redirect URL is required"))
	}

	return err
}

// Process post-processes the configuration after loading (before validation).
func (c *RedirectURLConfig) Process() error {
	if c.Signup == "" {
		c.Signup = c.Login
	}

	return nil
}

// CookieConfig contains auth cookie configuration.
type CookieConfig struct {
	Secure    bool
	Domain    string
	SetDomain bool
}

// Validate validates the configuration.
func (c CookieConfig) Validate() error {
	if c.SetDomain && c.Domain == "" {
		return errors.New("auth cookie domain is required")
	}

	return nil
}

// TokenConfig contains auth configuration.
type TokenConfig struct {
	SigningKey string
	Issuer     string
	Audience   string
}

// Validate validates the configuration.
func (c TokenConfig) Validate() error {
	var err error

	if c.SigningKey == "" {
		err = errors.Append(err, errors.New("auth token signing key is required"))
	}

	if len(c.SigningKey) < 32 {
		err = errors.Append(err, errors.New("auth token signing key must be at least 32 characters"))
	}

	if c.Issuer == "" {
		err = errors.Append(err, errors.New("auth token issuer is required"))
	}

	if c.Audience == "" {
		err = errors.Append(err, errors.New("auth token audience is required"))
	}

	return err
}

// RoleConfig contains role based authorization configuration.
type RoleConfig struct {
	Default string
	Binding map[string]string
}

// Validate validates the configuration.
func (c RoleConfig) Validate() error {
	if c.Default == "" {
		return errors.New("auth role default is required")
	}

	return nil
}
