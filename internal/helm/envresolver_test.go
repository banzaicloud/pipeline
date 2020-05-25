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

	"github.com/banzaicloud/pipeline/internal/common"
)

func Test_helm2EnvResolver_ResolveHelmEnv(t *testing.T) {
	type fields struct {
		helmHomesDir string
		orgService   OrgService
		logger       Logger
	}
	type args struct {
		ctx            context.Context
		organizationID uint
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       HelmEnv
		wantErr    bool
		setupMocks func(orgService *OrgService, arguments args)
	}{
		{
			name: "successfully resolve helm2 environment for orgID",
			fields: fields{
				helmHomesDir: "testHomesDir",
				orgService:   &MockOrgService{},
				logger:       common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 1,
			},
			want: HelmEnv{
				home:         "testHomesDir/testOrg/helm",
				platform:     false,
				cacheDir:     "testHomesDir/testOrg/cache",
				repoCacheDir: "",
			},
			wantErr: false,
			setupMocks: func(orgService *OrgService, arguments args) {
				orgServiceMock := (*orgService).(*MockOrgService)
				orgServiceMock.On("GetOrgNameByOrgID", arguments.ctx, arguments.organizationID).
					Return("testOrg", nil)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(&tt.fields.orgService, tt.args)
			h2r := helm2EnvResolver{
				envResolver{
					helmHomesDir: tt.fields.helmHomesDir,
					orgService:   tt.fields.orgService,
					logger:       tt.fields.logger,
				},
			}
			got, err := h2r.ResolveHelmEnv(tt.args.ctx, tt.args.organizationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveHelmEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveHelmEnv() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_helm3EnvResolver_ResolveHelmEnv(t *testing.T) {
	type fields struct {
		helmHomesDir string
		orgService   OrgService
		logger       Logger
	}
	type args struct {
		ctx            context.Context
		organizationID uint
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       HelmEnv
		wantErr    bool
		setupMocks func(orgService *OrgService, arguments args)
	}{
		{
			name: "successfully resolve helm3 environment for orgID",
			fields: fields{
				helmHomesDir: "testHomesDir",
				orgService:   &MockOrgService{},
				logger:       common.NoopLogger{},
			},
			args: args{
				ctx:            context.Background(),
				organizationID: 1,
			},
			want: HelmEnv{
				home:         "testHomesDir/testOrg/helm/repository/repositories.yaml",
				repoCacheDir: "testHomesDir/testOrg/helm/repository/cache",
				platform:     false,
				cacheDir:     "testHomesDir/testOrg/cache",
			},
			wantErr: false,
			setupMocks: func(orgService *OrgService, arguments args) {
				orgServiceMock := (*orgService).(*MockOrgService)
				orgServiceMock.On("GetOrgNameByOrgID", arguments.ctx, arguments.organizationID).
					Return("testOrg", nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(&tt.fields.orgService, tt.args)
			h3r := helm3EnvResolver{
				envResolver{
					helmHomesDir: tt.fields.helmHomesDir,
					orgService:   tt.fields.orgService,
					logger:       tt.fields.logger,
				},
			}
			got, err := h3r.ResolveHelmEnv(tt.args.ctx, tt.args.organizationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveHelmEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveHelmEnv() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_envResolver_ResolvePlatformEnv(t *testing.T) {
	type fields struct {
		helmHomesDir string
		orgService   OrgService
		logger       Logger
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    HelmEnv
		wantErr bool
	}{
		{
			name: "resolv the pipeline env",
			fields: fields{
				helmHomesDir: "testHomesDir",
				orgService:   &MockOrgService{},
				logger:       common.NoopLogger{},
			},
			args: args{
				ctx: context.Background(),
			},
			want: HelmEnv{
				home:     "testHomesDir-pipeline/helm",
				platform: true,
				cacheDir: "testHomesDir-pipeline/cache",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := envResolver{
				helmHomesDir: tt.fields.helmHomesDir,
				orgService:   tt.fields.orgService,
				logger:       tt.fields.logger,
			}
			got, err := er.ResolvePlatformEnv(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePlatformEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolvePlatformEnv() got = %v, want %v", got, tt.want)
			}
		})
	}
}
