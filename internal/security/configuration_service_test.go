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
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

func Test_configurationService_GetConfiguration(t *testing.T) {
	logger := commonadapter.NewLogger(logur.NewTestLogger())
	featureAdapterMock := MockFeatureAdapter{}
	featureAdapterMock.On("IsActive", context.Background(), uint(10), mock.Anything).Return(false, nil)
	featureAdapterMock.On("IsActive", context.Background(), uint(11), mock.Anything).Return(true, nil)

	featureAdapterMock.On("GetFeatureConfig", context.Background(), uint(11), mock.Anything).Return(Config{
		ApiEnabled: true,
		Enabled:    true,
		Endpoint:   "custom.anchore.com",
		AdminUser:  "",
		AdminPass:  "",
	}, nil)

	type fields struct {
		defaultConfig  Config
		featureAdapter FeatureAdapter
		logger         common.Logger
	}
	type args struct {
		ctx       context.Context
		clusterID uint
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "security scan api disabled",
			fields: fields{
				defaultConfig: Config{
					ApiEnabled: false,
				},
				featureAdapter: &featureAdapterMock,
				logger:         logger,
			},
			args: args{
				ctx:       context.Background(),
				clusterID: 10,
			},
			want:    Config{},
			wantErr: true,
		},
		{
			name: "security scan api enabled, feature is inactive, default config disabled",
			fields: fields{
				defaultConfig: Config{
					ApiEnabled: true,
				},
				featureAdapter: &featureAdapterMock,
				logger:         logger,
			},
			args: args{
				ctx:       context.Background(),
				clusterID: uint(10),
			},
			want:    Config{},
			wantErr: true,
		},
		{
			name: "security scan api enabled, feature is inactive, default config enabled",
			fields: fields{
				defaultConfig: Config{
					ApiEnabled: true,
					Enabled:    true,
					Endpoint:   "example.com",
				},
				featureAdapter: &featureAdapterMock,
				logger:         logger,
			},
			args: args{
				ctx:       context.Background(),
				clusterID: 10,
			},
			want: Config{
				ApiEnabled: true,
				Enabled:    true,
				Endpoint:   "example.com",
				AdminUser:  "",
				AdminPass:  "",
			},
			wantErr: false,
		},
		{
			name: "security scan api enabled, feature is inactive, default config enabled",
			fields: fields{
				defaultConfig: Config{
					ApiEnabled: true,
					Enabled:    true,
					Endpoint:   "example.com",
				},
				featureAdapter: &featureAdapterMock,
				logger:         logger,
			},
			args: args{
				ctx:       context.Background(),
				clusterID: 10,
			},
			want: Config{
				ApiEnabled: true,
				Enabled:    true,
				Endpoint:   "example.com",
				AdminUser:  "",
				AdminPass:  "",
			},
			wantErr: false,
		},
		{
			name: "security scan api enabled, feature is active, custom anchore enabled",
			fields: fields{
				defaultConfig: Config{
					ApiEnabled: true,
					Enabled:    true,
					Endpoint:   "example.com",
				},
				featureAdapter: &featureAdapterMock,
				logger:         logger,
			},
			args: args{
				ctx:       context.Background(),
				clusterID: 11,
			},
			want: Config{
				ApiEnabled: true,
				Enabled:    true,
				Endpoint:   "custom.anchore.com",
				AdminUser:  "",
				AdminPass:  "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigurationService(
				tt.fields.defaultConfig,
				tt.fields.featureAdapter,
				tt.fields.logger,
			)

			got, err := c.GetConfiguration(tt.args.ctx, tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfiguration() got = %v, want %v", got, tt.want)
			}
		})
	}
}
