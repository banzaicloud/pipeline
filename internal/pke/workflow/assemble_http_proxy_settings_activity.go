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
	OrganizationID     uint
	HTTPProxyHostPort  string
	HTTPProxySecretID  string
	HTTPProxyScheme    string
	HTTPSProxyHostPort string
	HTTPSProxySecretID string
	HTTPSProxyScheme   string
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
	var httpScheme = input.HTTPProxyScheme
	if httpScheme == "" {
		httpScheme = "http"
	}
	output.Settings.HTTPProxyURL, err = a.assembleProxyURL(ctx, httpScheme, input.HTTPProxyHostPort, input.OrganizationID, input.HTTPProxySecretID)
	if err != nil {
		return
	}

	var httpsScheme = input.HTTPSProxyScheme
	if httpsScheme == "" {
		httpsScheme = "https"
	}
	output.Settings.HTTPSProxyURL, err = a.assembleProxyURL(ctx, httpsScheme, input.HTTPSProxyHostPort, input.OrganizationID, input.HTTPSProxySecretID)
	if err != nil {
		return
	}

	return
}

func (a AssembleHTTPProxySettingsActivity) assembleProxyURL(
	ctx context.Context,
	scheme string,
	hostPort string,
	organizationID uint,
	secretID string,
) (string, error) {
	if hostPort == "" {
		return "", nil
	}

	var user *url.Userinfo
	if secretID != "" {
		s, err := a.secrets.GetSecret(ctx, organizationID, secretID)
		if err != nil {
			return "", errors.WrapIf(err, "failed to get secret")
		}
		user = url.UserPassword(s.Username(), s.Password())
	}

	return (&url.URL{
		Scheme: scheme,
		User:   user,
		Host:   hostPort,
	}).String(), nil
}
