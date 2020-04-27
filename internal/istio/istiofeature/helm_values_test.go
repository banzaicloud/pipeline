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

package istiofeature

import (
	"reflect"
	"testing"
)

func TestFromValues(t *testing.T) {
	expectedD := 123
	expectedE := true

	type A struct {
		B struct {
			C struct {
				D int  `json:"d,omitempty"`
				E bool `json:"e,omitempty"`
			} `json:"c,omitempty"`
		} `json:"b,omitempty"`
	}

	values := A{
		B: struct {
			C struct {
				D int  `json:"d,omitempty"`
				E bool `json:"e,omitempty"`
			} `json:"c,omitempty"`
		}{
			C: struct {
				D int  `json:"d,omitempty"`
				E bool `json:"e,omitempty"`
			}{
				D: expectedD,
				E: expectedE,
			},
		},
	}

	mapStringValues, err := ConvertStructure(values)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if b, ok := mapStringValues["b"].(map[string]interface{}); !ok {
		t.Fatalf("expected map[string]interface{} for field b")
	} else {
		if c, ok := b["c"].(map[string]interface{}); !ok {
			t.Fatalf("expected map[string]interface{} for field c")
		} else {
			if d, ok := c["d"]; !ok {
				t.Fatalf("missing field d")
			} else {
				if d != float64(expectedD) {
					t.Fatalf("invalid value for d %d type: %+v", d, reflect.TypeOf(d))
				}
			}
			if e, ok := c["e"]; !ok {
				t.Fatalf("missing field e")
			} else {
				if e != expectedE {
					t.Fatalf("invalid value for e %+v type: %+v", e, reflect.TypeOf(e))
				}
			}
		}
	}
}
