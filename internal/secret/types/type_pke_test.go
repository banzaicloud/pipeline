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

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/secret"
)

func TestPKEType(t *testing.T) {
	assert.Implements(t, (*secret.Type)(nil), new(PKEType))
	assert.Implements(t, (*secret.GeneratorType)(nil), new(PKEType))
	assert.Implements(t, (*secret.CleanupType)(nil), new(PKEType))
}

func TestPKEType_Validate(t *testing.T) {
	typ := PKEType{}

	assert.NoError(t, typ.Validate(nil))
}

func TestPKEType_ValidateNew(t *testing.T) {
	typ := PKEType{}

	complete, err := typ.ValidateNew(nil)

	assert.False(t, complete)
	assert.NoError(t, err)
}

func TestPKEType_Generate(t *testing.T) {
	// TODO
}

func TestPKEType_Cleanup(t *testing.T) {
	// TODO
}
