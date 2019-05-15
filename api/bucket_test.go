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

package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBucketNotFoundResponseCode(t *testing.T) {

	tests := []struct {
		name   string
		errMsg string
		code   int
	}{
		{
			name:   "response code should be 404",
			errMsg: "not found",
			code:   http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nfe := BucketNotFoundError{errMessage: test.errMsg}
			er := ErrorResponseFrom(nfe)

			assert.Equal(t, er.Code, test.code)
			assert.Equal(t, er.Message, test.errMsg)
		})
	}
}
