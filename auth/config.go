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
	"errors"
)

// Config contains auth configuration.
type Config struct {
	OIDC   OIDCConfig
	CLI    CLIConfig
	Cookie CookieConfig
	Token  TokenConfig
	Role   RoleConfig
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if err := c.OIDC.Validate(); err != nil {
		return err
	}

	if err := c.CLI.Validate(); err != nil {
		return err
	}

	if err := c.Cookie.Validate(); err != nil {
		return err
	}

	if err := c.Token.Validate(); err != nil {
		return err
	}

	if err := c.Role.Validate(); err != nil {
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
	if c.Issuer == "" {
		return errors.New("auth oidc issuer is required")
	}

	if c.ClientID == "" {
		return errors.New("auth oidc client ID is required")
	}

	if c.ClientSecret == "" {
		return errors.New("auth oidc client secret is required")
	}

	return nil
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
	if c.SigningKey == "" {
		return errors.New("auth token signing key is required")
	}

	if len(c.SigningKey) < 32 {
		return errors.New("auth token signing key must be at least 32 characters")
	}

	if c.Issuer == "" {
		return errors.New("auth token issuer is required")
	}

	if c.Audience == "" {
		return errors.New("auth token issuer is required")
	}

	return nil
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
