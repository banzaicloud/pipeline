// Copyright © 2020 Banzai Cloud
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

package helm

import (
	"context"
	"reflect"
	"testing"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

func Test_service_AddRepository(t *testing.T) {
	type fields struct {
		store         Store
		secretStore   SecretStore
		repoValidator RepoValidator
		envResolver   EnvResolver
		envService    EnvService
		logger        common.Logger
	}
	type args struct {
		ctx            context.Context
		organizationID uint
		repository     Repository
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		setupMocks func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args)
		wantErr    bool
	}{
		{
			name: "validation fails on the repo URL",
			fields: fields{
				store:         &MockStore{},
				secretStore:   &MockSecretStore{},
				envResolver:   &MockEnvResolver{},
				envService:    &MockEnvService{},
				repoValidator: NewHelmRepoValidator(),
				logger:        common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 1,
				repository: Repository{
					Name:             "test-repo",
					URL:              "failing-URL",
					PasswordSecretID: "password-ref",
				},
			},
			setupMocks: func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args) {
				secretStoreMock := (*secretStore).(*MockSecretStore)
				secretStoreMock.On("CheckPasswordSecret", arguments.ctx, arguments.repository.PasswordSecretID).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "validation fails on the password secret reference",
			fields: fields{
				store:         &MockStore{},
				secretStore:   &MockSecretStore{},
				envResolver:   &MockEnvResolver{},
				envService:    &MockEnvService{},
				repoValidator: NewHelmRepoValidator(),
				logger:        common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 1,
				repository: Repository{
					Name:             "test-repo",
					URL:              "https://example.com/charts",
					PasswordSecretID: "password-ref",
				},
			},
			setupMocks: func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args) {
				secretStoreMock := (*secretStore).(*MockSecretStore)
				secretStoreMock.On("CheckPasswordSecret", arguments.ctx, arguments.repository.PasswordSecretID).Return(errors.New("secret doesn't exist"))
			},
			wantErr: true,
		},
		{
			name: "helm repository already exists",
			fields: fields{
				store:         &MockStore{},
				secretStore:   &MockSecretStore{},
				envResolver:   &MockEnvResolver{},
				envService:    &MockEnvService{},
				repoValidator: NewHelmRepoValidator(),
				logger:        common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 1,
				repository: Repository{
					Name:             "test-repo",
					URL:              "https://example.com/charts",
					PasswordSecretID: "password-ref",
				},
			},
			setupMocks: func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args) {
				secretStoreMock := (*secretStore).(*MockSecretStore)
				secretStoreMock.On("CheckPasswordSecret", arguments.ctx, arguments.repository.PasswordSecretID).Return(nil)

				envResolverMock := (*envResolver).(*MockEnvResolver)
				envResolverMock.On("ResolveHelmEnv", arguments.ctx, arguments.organizationID).Return(HelmEnv{home: "/test"}, nil)

				envServiceMock := (*envService).(*MockEnvService)
				envServiceMock.On("ListRepositories", arguments.ctx, HelmEnv{home: "/test"}).Return([]Repository{
					{Name: "test-repo"},
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "helm repository successfully created",
			fields: fields{
				store:         &MockStore{},
				secretStore:   &MockSecretStore{},
				envResolver:   &MockEnvResolver{},
				envService:    &MockEnvService{},
				repoValidator: NewHelmRepoValidator(),
				logger:        common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 1,
				repository: Repository{
					Name:             "test-repo",
					URL:              "https://example.com/charts",
					PasswordSecretID: "password-ref",
				},
			},
			setupMocks: func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args) {
				secretStoreMock := (*secretStore).(*MockSecretStore)
				secretStoreMock.On("CheckPasswordSecret", arguments.ctx, arguments.repository.PasswordSecretID).Return(nil)

				storeMock := (*store).(*MockStore)
				storeMock.On("Get", arguments.ctx, arguments.organizationID, arguments.repository).Return(Repository{}, errors.New("repo not found"))
				storeMock.On("Create", arguments.ctx, arguments.organizationID, arguments.repository).Return(nil)

				envResolverMock := (*envResolver).(*MockEnvResolver)
				envResolverMock.On("ResolveHelmEnv", arguments.ctx, arguments.organizationID).Return(HelmEnv{home: "/test"}, nil)

				envServiceMock := (*envService).(*MockEnvService)
				envServiceMock.On("ListRepositories", arguments.ctx, HelmEnv{home: "/test"}).Return([]Repository{}, nil)
				envServiceMock.On("AddRepository", arguments.ctx, HelmEnv{home: "/test"}, arguments.repository).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(&tt.fields.store, &tt.fields.secretStore, &tt.fields.envResolver, &tt.fields.envService, tt.args)
			s := service{
				store:         tt.fields.store,
				secretStore:   tt.fields.secretStore,
				repoValidator: tt.fields.repoValidator,
				envResolver:   tt.fields.envResolver,
				envService:    tt.fields.envService,
				logger:        tt.fields.logger,
			}

			if err := s.AddRepository(tt.args.ctx, tt.args.organizationID, tt.args.repository); (err != nil) != tt.wantErr {
				t.Errorf("AddRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_service_ListRepositories(t *testing.T) {
	type fields struct {
		store         Store
		secretStore   SecretStore
		repoValidator RepoValidator
		envResolver   EnvResolver
		envService    EnvService
		logger        Logger
	}
	type args struct {
		ctx            context.Context
		organizationID uint
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantRepos  []Repository
		setupMocks func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args)
		wantErr    bool
	}{
		{
			name: "list default repositories",
			fields: fields{
				store:         &MockStore{},
				secretStore:   &MockSecretStore{},
				repoValidator: NewHelmRepoValidator(),
				envResolver:   &MockEnvResolver{},
				envService:    &MockEnvService{},
				logger:        common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 2,
			},
			wantRepos: []Repository{
				{
					Name: "stable",
					URL:  "https://kubernetes-charts.storage.googleapis.com",
				},
				{
					Name: "banzaicloud-stable",
					URL:  "https://kubernetes-charts.banzaicloud.com",
				},
				{
					Name: "loki",
					URL:  "https://grafana.github.io/loki/charts",
				},
			},
			setupMocks: func(store *Store, secretStore *SecretStore, envResolver *EnvResolver, envService *EnvService, arguments args) {
				storeMock := (*store).(*MockStore)
				storeMock.On("List", arguments.ctx, arguments.organizationID).Return([]Repository{}, nil)

				envResolverMock := (*envResolver).(*MockEnvResolver)
				envResolverMock.On("ResolveHelmEnv", arguments.ctx, arguments.organizationID).Return(HelmEnv{home: "/test"}, nil)

				envServiceMock := (*envService).(*MockEnvService)
				envServiceMock.On("ListRepositories", arguments.ctx, HelmEnv{home: "/test"}).Return(
					[]Repository{
						{
							Name: "stable",
							URL:  "https://kubernetes-charts.storage.googleapis.com",
						},
						{
							Name: "banzaicloud-stable",
							URL:  "https://kubernetes-charts.banzaicloud.com",
						},
						{
							Name: "loki",
							URL:  "https://grafana.github.io/loki/charts",
						},
					},
					nil,
				)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(&tt.fields.store, &tt.fields.secretStore, &tt.fields.envResolver, &tt.fields.envService, tt.args)
			s := service{
				store:         tt.fields.store,
				secretStore:   tt.fields.secretStore,
				repoValidator: tt.fields.repoValidator,
				envResolver:   tt.fields.envResolver,
				envService:    tt.fields.envService,
				logger:        tt.fields.logger,
			}
			gotRepos, err := s.ListRepositories(tt.args.ctx, tt.args.organizationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRepositories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRepos, tt.wantRepos) {
				t.Errorf("ListRepositories() gotRepos = %v, want %v", gotRepos, tt.wantRepos)
			}
		})
	}
}
