// Copyright © 2018 Banzai Cloud
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

package secret_test

import (
	"testing"

	"github.com/banzaicloud/pipeline/src/secret"
)

func TestRandomString(t *testing.T) {

	cases := []struct {
		name    string
		genType string
		length  int
		isError bool
	}{
		{name: "randAlpha", genType: "randAlpha", length: 12, isError: false},
		{name: "randAlphaNum", genType: "randAlphaNum", length: 13, isError: false},
		{name: "randNumeric", genType: "randNumeric", length: 14, isError: false},
		{name: "randAscii", genType: "randAscii", length: 99, isError: false},
		{name: "Wrong Type", genType: "randAlha", length: 0, isError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := secret.RandomString(tc.name, tc.length)
			if err != nil {
				if !tc.isError {
					t.Errorf("Error occours: %s", err.Error())
				}
			} else if tc.isError {
				t.Errorf("Not occours error")
			}
			if len(result) != tc.length {
				t.Errorf("result length mismatch")
			}

		})
	}

}
