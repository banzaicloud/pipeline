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

package services

import (
	"testing"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
)

func TestBindIntegratedServiceSpec(t *testing.T) {
	type args struct {
		inputSpec integratedservices.IntegratedServiceSpec
		boundSpec interface{}
	}

	esp := expiry.ServiceSpec{}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "expiry spec successfully bound",
			args: args{
				inputSpec: map[string]interface{}{
					"date": "2020-01-09T12:42:00Z",
				},
				boundSpec: &esp,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := BindIntegratedServiceSpec(tt.args.inputSpec, tt.args.boundSpec); (err != nil) != tt.wantErr {
				t.Errorf("BindIntegratedServiceSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
