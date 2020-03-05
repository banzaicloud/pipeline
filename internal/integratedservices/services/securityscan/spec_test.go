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

package securityscan

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_webHookConfigSpec_GetValues(t *testing.T) {
	type fields struct {
		Enabled    bool
		Selector   string
		Namespaces []string
	}
	tests := []struct {
		name   string
		fields fields
		want   ImageValidatorChartValues
	}{
		{
			name: "include all namespaces / default configuration",
			fields: fields{
				Enabled:    true,
				Selector:   selectorInclude,
				Namespaces: []string{selectedAllStar},
			},
			want: ImageValidatorChartValues{
				NamespaceSelector: nil,
				ObjectSelector:    nil,
			}, // empty values!
		},
		{
			name: "include some namespaces", // namespaces labeled with scan=scan
			fields: fields{
				Enabled:    true,
				Selector:   selectorInclude,
				Namespaces: []string{"ns1", "ns2"},
			},
			want: ImageValidatorChartValues{
				NamespaceSelector: &SetBasedSelector{
					MatchLabels: map[string]string{labelKey: "scan"},
				},
				ObjectSelector: nil,
			},
		},
		{
			name: "exclude all namespaces", // labels removed from namespaces
			fields: fields{
				Enabled:    true,
				Selector:   selectorExclude,
				Namespaces: []string{selectedAllStar},
			},
			want: ImageValidatorChartValues{
				NamespaceSelector: &SetBasedSelector{
					MatchLabels: map[string]string{labelKey: "scan"},
				},
				ObjectSelector: nil,
			},
		},
		{
			name: "exclude some namespaces", // relevant namespaces labeled with scan=noscan
			fields: fields{
				Enabled:    true,
				Selector:   selectorExclude,
				Namespaces: []string{"ns1", "ns2"},
			},
			want: ImageValidatorChartValues{
				NamespaceSelector: nil,
				ObjectSelector:    nil,
			}, // empty values here / namespaces labeled, the default config applies
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := webHookConfigSpec{
				Enabled:    tt.fields.Enabled,
				Selector:   tt.fields.Selector,
				Namespaces: tt.fields.Namespaces,
			}
			if got := w.GetValues(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalValues(t *testing.T) {
	tests := []struct {
		name      string
		whCfgSpec webHookConfigSpec
	}{
		{
			whCfgSpec: webHookConfigSpec{
				Enabled:    true,
				Selector:   selectorInclude,
				Namespaces: []string{"ns1", "ns2"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vals := test.whCfgSpec.GetValues()
			valuesBytes, _ := json.Marshal(vals)

			t.Log(string(valuesBytes))
		})
	}
}
