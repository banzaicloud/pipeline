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
	configService ConfigurationService
	logger        common.Logger
}

func MakeAnchoreClient(cfgService ConfigurationService, logger common.Logger) AnchoreClient {
	return anchoreClient{
		configService: cfgService,
		logger:        logger,
	}
}

func (a anchoreClient) CreateAccount(ctx context.Context, accountName string, email string) error {
	panic("implement me")
}

func (a anchoreClient) CreateUser(ctx context.Context, userName string, password string) error {
	panic("implement me")
}

func (a anchoreClient) GetUser(ctx context.Context, userName string) (interface{}, error) {
	panic("implement me")
}

func (a anchoreClient) GetUserCreadentials(ctx context.Context, userName string) (string, error) {
	panic("implement me")
}
