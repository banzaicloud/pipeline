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
	"bytes"
	"crypto/rand"
	"io"
	"math/big"

	"emperror.dev/errors"
)

type PasswordGenerator struct {
	IndexGenerator IndexGenerator
}

func NewCryptoPasswordGenerator() PasswordGenerator {
	return PasswordGenerator{
		IndexGenerator: UniformRandomIndexGenerator{
			Random: rand.Reader,
		},
	}
}

func (g PasswordGenerator) GenerateAlphabetic(length int) (string, error) {
	return g.generate(alphabeticRunes, length)
}

func (g PasswordGenerator) GenerateAlphanumeric(length int) (string, error) {
	return g.generate(alphanumericRunes, length)
}

func (g PasswordGenerator) GenerateASCII(length int) (string, error) {
	return g.generate(asciiRunes, length)
}

func (g PasswordGenerator) GenerateNumeric(length int) (string, error) {
	return g.generate(numericRunes, length)
}

// nolint: gochecknoglobals
var (
	alphabeticRunes   = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	alphanumericRunes = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	asciiRunes        = []rune(" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~")
	numericRunes      = []rune("0123456789")
)

func (g PasswordGenerator) generate(alphabet []rune, length int) (string, error) {
	if length < 0 {
		return "", errors.New("length must be a non-negative number")
	}

	l := len(alphabet)

	var b bytes.Buffer
	for i := 0; i < length; i++ {
		idx, err := g.IndexGenerator.Generate(l)
		if err != nil {
			return "", errors.WrapIf(err, "failed to generate index")
		}
		r := alphabet[idx]
		b.WriteRune(r)
	}
	return b.String(), nil
}

type IndexGenerator interface {
	Generate(limit int) (int, error)
}

type UniformRandomIndexGenerator struct {
	Random io.Reader
}

func (g UniformRandomIndexGenerator) Generate(limit int) (int, error) {
	idx, err := rand.Int(g.Random, big.NewInt(int64(limit)))
	return int(idx.Int64()), err
}
