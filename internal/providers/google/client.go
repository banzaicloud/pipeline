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

package google

import (
	"context"
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/serviceusage/v1"
)

// ClientFactory creates a new HTTP client for Google API communication.
type ClientFactory struct {
	secretStore SecretStore
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(secretStore SecretStore) ClientFactory {
	return ClientFactory{
		secretStore: secretStore,
	}
}

// CreateClient creates a new HTTP client for Google API communication.
func (c ClientFactory) CreateClient(ctx context.Context, secretID string) (*http.Client, error) {
	return c.CreateClientWithScopes(ctx, secretID)
}

// CreateClientWithScopes creates a new HTTP client for Google API communication.
func (c ClientFactory) CreateClientWithScopes(ctx context.Context, secretID string, scopes ...string) (*http.Client, error) {
	secret, err := c.secretStore.GetSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	if len(scopes) == 0 {
		// This is here for backward compatibility, but it should probably be explicitly stated everywhere
		scopes = []string{serviceusage.CloudPlatformScope}
	}

	jsonSecret, err := json.Marshal(secret)
	if err != nil {
		return nil, err
	}

	config, err := google.JWTConfigFromJSON(jsonSecret, scopes...)
	if err != nil {
		return nil, err
	}

	return config.Client(ctx), nil
}
