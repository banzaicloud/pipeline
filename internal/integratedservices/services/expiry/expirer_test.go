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

package expiry

import (
	"context"
	"testing"
	"time"

	"github.com/banzaicloud/pipeline/internal/common"
)

func Test_syncExpirer_Expire(t *testing.T) {
	type fields struct {
		logger common.Logger
	}
	type args struct {
		ctx       context.Context
		clusterID uint
		date      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "expirer should execute at the defined date",
			fields: fields{
				logger: common.NewNoopLogger(),
			},
			args: args{
				ctx:  context.Background(),
				date: time.Now().Add(15 * time.Second).Format(time.RFC3339),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := syncExpirer{
				logger: tt.fields.logger,
			}

			if err := s.Expire(tt.args.ctx, tt.args.clusterID, tt.args.date); (err != nil) != tt.wantErr {
				t.Errorf("Expire() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test(t *testing.T) {

	dateX := time.Now().Format(time.RFC3339)
	dateY, _ := time.Parse(time.RFC3339, "2020-01-13T16:56:36+05:00")
	dateZ, _ := time.ParseInLocation(time.RFC3339, "2020-01-13T16:56:36+05:00", time.Now().Location())

	t.Logf("dateX: %s", dateX)
	t.Logf("dateY: %s", dateY)
	t.Logf("dateZ: %s", dateZ)
}
