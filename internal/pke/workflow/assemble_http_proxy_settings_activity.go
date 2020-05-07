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

package workflow

import (
	"context"
	"net/url"

	"emperror.dev/errors"
)

const AssembleHTTPProxySettingsActivityName = "assemble-http-proxy-settings"

type PasswordSecret interface {
	Username() string
	Password() string
}

type PasswordSecretStore interface {
	GetSecret(ctx context.Context, organizationID uint, secretID string) (PasswordSecret, error)
}

type AssembleHTTPProxySettingsActivity struct {
	secrets PasswordSecretStore
}

func NewAssembleHTTPProxySettingsActivity(secrets PasswordSecretStore) AssembleHTTPProxySettingsActivity {
	return AssembleHTTPProxySettingsActivity{
		secrets: secrets,
	}
}

type AssembleHTTPProxySettingsActivityInput struct {
	OrganizationID uint
	HTTP           ProxyOptions
	HTTPS          ProxyOptions
}

type ProxyOptions struct {
	// Deprecated: use URL field instead
	HostPort string
	SecretID string
	// Deprecated: use URL field instead
	Scheme string
	URL    string
}

type AssembleHTTPProxySettingsActivityOutput struct {
	Settings HTTPProxy
}

type HTTPProxy struct {
	HTTPProxyURL  string
	HTTPSProxyURL string
}

func (a AssembleHTTPProxySettingsActivity) Execute(
	ctx context.Context,
	input AssembleHTTPProxySettingsActivityInput,
) (output AssembleHTTPProxySettingsActivityOutput, err error) {
	if input.HTTP.Scheme == "" {
		input.HTTP.Scheme = "http"
	}

	output.Settings.HTTPProxyURL, err = a.assembleProxyURL(ctx, input.OrganizationID, input.HTTP)
	if err != nil {
		return
	}

	if input.HTTPS.Scheme == "" {
		input.HTTPS.Scheme = "https"
	}
	output.Settings.HTTPSProxyURL, err = a.assembleProxyURL(ctx, input.OrganizationID, input.HTTPS)
	if err != nil {
		return
	}

	return
}

func (a AssembleHTTPProxySettingsActivity) assembleProxyURL(
	ctx context.Context,
	organizationID uint,
	options ProxyOptions,
) (string, error) {
	var proxyURL *url.URL
	var user *url.Userinfo

	// get user info
	if options.SecretID != "" {
		s, err := a.secrets.GetSecret(ctx, organizationID, options.SecretID)
		if err != nil {
			return "", errors.WrapIf(err, "failed to get secret")
		}
		user = url.UserPassword(s.Username(), s.Password())
	}

	if options.URL == "" {
		if options.HostPort == "" {
			return "", nil
		}

		proxyURL = &url.URL{
			Scheme: options.Scheme,
			Host:   options.HostPort,
		}
	} else {
		var err error
		proxyURL, err = url.Parse(options.URL)
		if err != nil {
			return "", errors.WrapIfWithDetails(err, "failed to parse proxy url", "url", options.URL)
		}
	}

	proxyURL.User = user
	return proxyURL.String(), nil
}
