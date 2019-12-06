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

package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlphabetsAreShort(t *testing.T) {
	assert.LessOrEqual(t, len(alphabeticRunes), 256, "alphabeticRunes cannot be indexed by 1 byte")
	assert.LessOrEqual(t, len(alphanumericRunes), 256, "alphanumericRunes cannot be indexed by 1 byte")
	assert.LessOrEqual(t, len(asciiRunes), 256, "asciiRunes cannot be indexed by 1 byte")
	assert.LessOrEqual(t, len(numericRunes), 256, "numericRunes cannot be indexed by 1 byte")
}

func TestPasswordGenerator_GenerateAlphabetic(t *testing.T) {
	gen := PasswordGenerator{
		IndexGenerator: &dummyIndexGenerator{},
	}

	res, err := gen.GenerateAlphabetic(len(alphabeticRunes))
	require.NoError(t, err)
	assert.Equal(t, string(alphabeticRunes), res)
}

func TestPasswordGenerator_GenerateAlphanumeric(t *testing.T) {
	gen := PasswordGenerator{
		IndexGenerator: &dummyIndexGenerator{},
	}

	res, err := gen.GenerateAlphanumeric(len(alphanumericRunes))
	require.NoError(t, err)
	assert.Equal(t, string(alphanumericRunes), res)
}

func TestPasswordGenerator_GenerateASCII(t *testing.T) {
	gen := PasswordGenerator{
		IndexGenerator: &dummyIndexGenerator{},
	}

	res, err := gen.GenerateASCII(len(asciiRunes))
	require.NoError(t, err)
	assert.Equal(t, string(asciiRunes), res)
}

func TestPasswordGenerator_GenerateNumeric(t *testing.T) {
	gen := PasswordGenerator{
		IndexGenerator: &dummyIndexGenerator{},
	}

	res, err := gen.GenerateAlphanumeric(len(numericRunes))
	require.NoError(t, err)
	assert.Equal(t, string(numericRunes), res)
}

type dummyIndexGenerator struct {
	i int
}

func (g *dummyIndexGenerator) Generate(limit int) (int, error) {
	if g.i >= limit {
		g.i = 0
	}
	idx := g.i
	g.i++
	return idx, nil
}
