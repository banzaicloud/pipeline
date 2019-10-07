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
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
)

func TestAssembleHTTPProxySettings(t *testing.T) {
	orgID := uint(13)
	httpSecretID := "http-secret-id"
	httpsSecretID := "https-secret-id"

	a := MakeAssembleHTTPProxySettingsActivity(dummyPasswordSecretStore{
		entries: map[uint]map[string]dummyPasswordSecret{
			orgID: map[string]dummyPasswordSecret{
				httpSecretID: dummyPasswordSecret{
					username: "http-username",
					password: "http-password",
				},
				httpsSecretID: dummyPasswordSecret{
					username: "https-username",
					password: "https-password",
				},
			},
		},
	})

	tc := map[string]struct {
		Input  AssembleHTTPProxySettingsActivityInput
		Output HTTPProxy
		Error  interface{}
	}{
		"with everything": {
			Input: AssembleHTTPProxySettingsActivityInput{
				OrganizationID:     orgID,
				HTTPProxyHostPort:  "http-proxy.org:1234",
				HTTPProxySecretID:  httpSecretID,
				HTTPSProxyHostPort: "https-proxy.org:2345",
				HTTPSProxySecretID: httpsSecretID,
			},
			Output: HTTPProxy{
				HTTPProxy:  "http://http-username:http-password@http-proxy.org:1234",
				HTTPSProxy: "https://https-username:https-password@https-proxy.org:2345",
			},
		},
		"http only with secret": {
			Input: AssembleHTTPProxySettingsActivityInput{
				OrganizationID:    orgID,
				HTTPProxyHostPort: "http-proxy.org:1234",
				HTTPProxySecretID: httpSecretID,
			},
			Output: HTTPProxy{
				HTTPProxy: "http://http-username:http-password@http-proxy.org:1234",
			},
		},
		"http only without secret": {
			Input: AssembleHTTPProxySettingsActivityInput{
				OrganizationID:    orgID,
				HTTPProxyHostPort: "http-proxy.org:1234",
			},
			Output: HTTPProxy{
				HTTPProxy: "http://http-proxy.org:1234",
			},
		},
		"https only with secret": {
			Input: AssembleHTTPProxySettingsActivityInput{
				OrganizationID:     orgID,
				HTTPSProxyHostPort: "https-proxy.org:2345",
				HTTPSProxySecretID: httpsSecretID,
			},
			Output: HTTPProxy{
				HTTPSProxy: "https://https-username:https-password@https-proxy.org:2345",
			},
		},
		"https only without secret": {
			Input: AssembleHTTPProxySettingsActivityInput{
				OrganizationID:     orgID,
				HTTPSProxyHostPort: "https-proxy.org:2345",
			},
			Output: HTTPProxy{
				HTTPSProxy: "https://https-proxy.org:2345",
			},
		},
		"empty": {
			Input:  AssembleHTTPProxySettingsActivityInput{},
			Output: HTTPProxy{},
		},
		"missing secret": {
			Input: AssembleHTTPProxySettingsActivityInput{
				OrganizationID:     uint(24),
				HTTPProxyHostPort:  "http-proxy.org:1234",
				HTTPProxySecretID:  httpSecretID,
				HTTPSProxyHostPort: "https-proxy.org:2345",
				HTTPSProxySecretID: httpsSecretID,
			},
			Error: true,
		},
	}
	for name, tv := range tc {
		t.Run(name, func(t *testing.T) {
			output, err := a.Execute(context.Background(), tv.Input)
			switch tv.Error {
			case true:
				require.Error(t, err)
			case false, nil:
				require.NoError(t, err)
				require.Equal(t, AssembleHTTPProxySettingsActivityOutput{
					Settings: tv.Output,
				}, output)
			default:
				require.Equal(t, tv.Error, err)
			}
		})
	}
}

type dummyPasswordSecretStore struct {
	entries map[uint]map[string]dummyPasswordSecret
}

func (s dummyPasswordSecretStore) GetSecret(ctx context.Context, orgID uint, secretID string) (interface {
	Username() string
	Password() string
}, error) {
	if secrets, ok := s.entries[orgID]; ok {
		if secret, ok := secrets[secretID]; ok {
			return secret, nil
		}
	}
	return nil, errors.New("secret not found")
}

type dummyPasswordSecret struct {
	username string
	password string
}

func (s dummyPasswordSecret) Username() string {
	return s.username
}

func (s dummyPasswordSecret) Password() string {
	return s.password
}
