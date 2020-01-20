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
	"testing"
	"time"
)

func TestServiceSpec_Validate(t *testing.T) {
	type fields struct {
		Date string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "expiry date must be in RFC3339 format ",
			fields: fields{
				Date: "Mon Jan _2 15:04:05 MST 2006",
			},
			wantErr: true,
		},
		{
			name: "expiry date must be in the future",
			fields: fields{
				Date: "2006-01-02T15:04:05Z",
			},
			wantErr: true,
		},
		{
			name: "expiry date in RFC3339 and in the future",
			fields: fields{
				Date: time.Now().Add(60 * time.Minute).Format(time.RFC3339),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := ServiceSpec{
				Date: tt.fields.Date,
			}
			if err := s.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
