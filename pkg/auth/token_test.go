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

package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type idGeneratorStub struct {
	id string
}

func (s idGeneratorStub) Generate() string {
	return s.id
}

type clockStub struct {
	now time.Time
}

func (s clockStub) Now() time.Time {
	return s.now
}

func TestTokenGenerator_GenerateToken(t *testing.T) {
	now := time.Date(2019, time.September, 20, 14, 44, 00, 00, time.UTC)

	generator := NewTokenGenerator(
		"issuer",
		"audience",
		"signingKey",
		TokenIDGenerator(idGeneratorStub{"id"}),
		TokenGeneratorClock(clockStub{now}),
	)

	tokenID, signedToken, err := generator.GenerateToken("user", NoExpiration, "token", "my_text")
	require.NoError(t, err)

	const expectedSignedToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJhdWRpZW5jZSIsImp0aSI6ImlkIiwiaWF0IjoxNTY4OTkwNjQwLCJpc3MiOiJpc3N1ZXIiLCJzdWIiOiJ1c2VyIiwic2NvcGUiOiJhcGk6aW52b2tlIiwidHlwZSI6InRva2VuIiwidGV4dCI6Im15X3RleHQifQ.MmDG43-5P0H-o9yP3I4SXhinAuauXj27K2b4DazmmIs"

	assert.Equal(t, "id", tokenID)
	assert.Equal(t, expectedSignedToken, signedToken)
}
