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

package correlation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithID(t *testing.T) {
	ctx := WithID(context.Background(), "id")

	assert.Equal(t, "id", ctx.Value(correlationID))
}

func TestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), correlationID, "id")

	id, ok := ID(ctx)
	require.True(t, ok)
	assert.Equal(t, "id", id)
}
