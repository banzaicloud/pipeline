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
	"fmt"
	"strconv"
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

const Password = "password"

const (
	FieldPasswordUsername = "username"
	FieldPasswordPassword = "password"
)

type PasswordType struct{}

func (PasswordType) Name() string {
	return Password
}

func (PasswordType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldPasswordUsername, Required: true, IsSafeToDisplay: true, Description: "Your username"},
			{Name: FieldPasswordPassword, Required: false, Description: "Your password"},
		},
	}
}

// Note: this will only require the username field.
func (t PasswordType) Validate(data map[string]string) error {
	var violations []string

	if _, ok := data[FieldPasswordUsername]; !ok {
		violations = append(violations, fmt.Sprintf("missing key: %s", FieldPasswordUsername))
	}

	if _, ok := data[FieldPasswordPassword]; !ok {
		violations = append(violations, fmt.Sprintf("missing key: %s", FieldPasswordPassword))
	}

	if len(violations) > 0 {
		// For backward compatibility reasons, return the first violation as message
		return secret.NewValidationError(violations[0], violations)
	}

	return nil
}

func (t PasswordType) ValidateNew(data map[string]string) (bool, error) {
	if _, ok := data[FieldPasswordUsername]; !ok {
		violation := fmt.Sprintf("missing key: %s", FieldPasswordUsername)

		return false, secret.NewValidationError(violation, []string{violation})
	}

	if password, ok := data[FieldPasswordPassword]; !ok || password == "" {
		return false, nil
	} else if ok && len(strings.Split(password, ",")) == 2 { // TODO: passwords containing one comma will trigger generation!!!!!
		return false, nil
	}

	return true, nil
}

const defaultPasswordFormat = "randAlphaNum,12"

func (t PasswordType) Generate(_ uint, _ string, data map[string]string, _ []string) (map[string]string, error) {
	password := data[FieldPasswordPassword]
	if password == "" {
		password = defaultPasswordFormat
	}

	methodAndLength := strings.Split(password, ",")
	if len(methodAndLength) == 2 {
		length, err := strconv.Atoi(methodAndLength[1])
		if err != nil {
			return nil, errors.WrapIf(err, "failed to determine password length")
		}

		password, err := passwordRandomString(methodAndLength[0], length)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to generate password")
		}

		data[FieldPasswordPassword] = password
	}

	return data, nil
}

// passwordRandomString creates a random string whose length is the number of characters specified.
// TODO: reuse random function (or use single, struct level password generator in the type?).
func passwordRandomString(genType string, length int) (res string, err error) {
	gen := pkgSecret.NewCryptoPasswordGenerator()

	switch genType {
	case "randAlphaNum":
		return gen.GenerateAlphanumeric(length)
	case "randAlpha":
		return gen.GenerateAlphabetic(length)
	case "randNumeric":
		return gen.GenerateNumeric(length)
	case "randAscii":
		return gen.GenerateASCII(length)
	default:
		return "", errors.Errorf("unsupported random type: %s", genType)
	}
}
