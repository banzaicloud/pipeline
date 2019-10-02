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

package anchore

import (
	"context"
	"net/http"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/.gen/anchore"
	"github.com/banzaicloud/pipeline/internal/common"
)

// AnchoreClient defines Anchore operations
type AnchoreClient interface {
	CreateAccount(ctx context.Context, accountName string, email string) error
	CreateUser(ctx context.Context, userName string, password string) error
	GetUser(ctx context.Context, userName string) (interface{}, error)
	GetUserCreadentials(ctx context.Context, userName string) (string, error)
}

type anchoreClient struct {
	config Config
	logger common.Logger
}

func MakeAnchoreClient(cfg Config, logger common.Logger) AnchoreClient {
	return anchoreClient{
		config: cfg,
		logger: logger,
	}
}

func (a anchoreClient) CreateAccount(ctx context.Context, accountName string, email string) error {
	panic("implement me")
}

func (a anchoreClient) CreateUser(ctx context.Context, userName string, password string) error {
	panic("implement me")
}

func (a anchoreClient) GetUser(ctx context.Context, userName string) (interface{}, error) {
	usr, resp, err := a.getRestClient().UserManagementApi.GetAccountUser(a.authorizedContext(ctx), userName, userName)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to retrieve user from anchore")

		return nil, errors.WrapIf(err, "failed to retrieve user from anchore")
	}

	return usr, nil
}

func (a anchoreClient) GetUserCreadentials(ctx context.Context, userName string) (string, error) {
	credentials, resp, err := a.getRestClient().UserManagementApi.ListUserCredentials(a.authorizedContext(ctx), userName, userName)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to retrieve user from anchore")

		return "", errors.WrapIf(err, "failed to retrieve user from anchore")
	}

	for _, credential := range credentials {
		if credential.Value != "" {
			return credential.Value, nil
		}
	}

	return "", errors.NewWithDetails("no credentials found", "userName", userName)
}

func (a anchoreClient) authorizedContext(ctx context.Context) context.Context {

	basicAuth := anchore.BasicAuth{
		UserName: a.config.AdminUser,
		Password: a.config.AdminPass,
	}

	return context.WithValue(ctx, anchore.ContextBasicAuth, basicAuth)
}

func (a anchoreClient) getRestClient() *anchore.APIClient {

	return anchore.NewAPIClient(&anchore.Configuration{
		BasePath:      a.config.Endpoint,
		DefaultHeader: make(map[string]string),
		UserAgent:     "Pipeline/go",
	})

}
